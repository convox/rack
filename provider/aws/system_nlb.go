package aws

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
)

// validateNLBAllowCIDRParam applies the list-shape rules to
// NLBAllowCIDR / NLBInternalAllowCIDR: at most 5 non-empty entries, each a
// valid IPv4 CIDR (octets 0-255, mask 0-32), no duplicates, no leading/
// trailing whitespace (CF `Fn::Split` does not trim, so spaces in stored
// values slip through to AWS and surface as opaque CF errors). Empty input
// returns nil — blank is valid, and for the internal variant selects the
// VPCCIDR fallback in CF.
func validateNLBAllowCIDRParam(paramName, value string) error {
	if value == "" {
		return nil
	}
	entries := strings.Split(value, ",")
	seen := map[string]bool{}
	nonEmpty := 0
	for _, e := range entries {
		if e == "" {
			continue
		}
		trimmed := strings.TrimSpace(e)
		if trimmed == "" {
			continue
		}
		if trimmed != e {
			return fmt.Errorf("%s entry %q has leading or trailing whitespace; remove the spaces and retry", paramName, e)
		}
		ip, ipnet, err := net.ParseCIDR(trimmed)
		if err != nil {
			return fmt.Errorf("%s entry %q is not a valid IPv4 CIDR", paramName, trimmed)
		}
		if ip.To4() == nil {
			return fmt.Errorf("%s entry %q is not a valid IPv4 CIDR (IPv6 not supported)", paramName, trimmed)
		}
		// Guard against non-canonical forms like "10.0.0.1/24" (host bits set);
		// normalize to the network form and compare.
		if ipnet.String() != trimmed {
			return fmt.Errorf("%s entry %q is not canonical; use %q instead", paramName, trimmed, ipnet.String())
		}
		if seen[trimmed] {
			return fmt.Errorf("%s contains duplicate entry: %s", paramName, trimmed)
		}
		seen[trimmed] = true
		nonEmpty++
	}
	if nonEmpty > 5 {
		return fmt.Errorf("%s accepts at most 5 CIDRs; got %d", paramName, nonEmpty)
	}
	return nil
}

// appsUsingNLBScheme returns the list of gen2 apps on the rack whose current running release
// declares at least one service with an NLB port of the given scheme ("public" or "internal").
// Returns entries formatted as "app/service".
//
// Fails open on per-app errors (logs a warning and continues) so a single broken release does
// not block rack-wide operations.
func (p *Provider) appsUsingNLBScheme(scheme string) ([]string, error) {
	log := p.logger("appsUsingNLBScheme")

	apps, err := p.AppList()
	if err != nil {
		return nil, err
	}

	var blockers []string
	for _, a := range apps {
		if a.Tags["Generation"] != "2" {
			continue
		}
		if a.Release == "" {
			continue
		}
		r, err := p.ReleaseGet(a.Name, a.Release)
		if err != nil {
			log.Logf("warning: skipped app=%s err=%v", a.Name, err)
			continue
		}
		if r.Manifest == "" {
			continue
		}
		m, err := manifest.Load([]byte(r.Manifest), nil)
		if err != nil {
			log.Logf("warning: skipped app=%s manifest parse failed err=%v", a.Name, err)
			continue
		}
		for _, s := range m.Services {
			for _, np := range s.NLB {
				if np.Scheme == scheme {
					blockers = append(blockers, fmt.Sprintf("%s/%s", a.Name, s.Name))
					break
				}
			}
		}
	}
	return blockers, nil
}

// yesNo maps a bool to "Yes"/"No" — package-level so provider/aws can share
// the conversion between validator, render-context plumbing, and ad-hoc callers.
func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// validateNLBParams enforces the NLB rack-param rules:
//   - NLB=Yes incompatible with InternalOnly=Yes
//   - NLBInternal=Yes requires Internal=Yes (either already set or set in the same call)
//   - Disabling NLB/NLBInternal requires no dependent gen2 apps
//   - Allowlist CIDR params shape/limit/dedup
//   - Deletion protection + NLB=No interlock
//   - Customer-InstanceSecurityGroup incompatible with preserve_client_ip
func (p *Provider) validateNLBParams(opts structs.SystemUpdateOptions) error {
	want := func(key string) (string, bool) {
		if opts.Parameters == nil {
			return "", false
		}
		v, ok := opts.Parameters[key]
		return v, ok
	}

	currentNLB := yesNo(p.NLB)
	currentNLBInternal := yesNo(p.NLBInternal)
	currentInternal := yesNo(p.Internal)
	currentInternalOnly := yesNo(p.InternalOnly)

	if v, ok := want("NLBAllowCIDR"); ok {
		if err := validateNLBAllowCIDRParam("NLBAllowCIDR", v); err != nil {
			return err
		}
	}
	if v, ok := want("NLBInternalAllowCIDR"); ok {
		if err := validateNLBAllowCIDRParam("NLBInternalAllowCIDR", v); err != nil {
			return err
		}
	}

	nextNLB, nlbSet := want("NLB")
	if !nlbSet {
		nextNLB = currentNLB
	}
	nextNLBInternal, nlbiSet := want("NLBInternal")
	if !nlbiSet {
		nextNLBInternal = currentNLBInternal
	}
	nextInternal, internalSet := want("Internal")
	if !internalSet {
		nextInternal = currentInternal
	}
	nextInternalOnly, _ := want("InternalOnly")
	if nextInternalOnly == "" {
		nextInternalOnly = currentInternalOnly
	}

	if nextNLB == "Yes" && nextInternalOnly == "Yes" {
		return fmt.Errorf("cannot enable public NLB on an InternalOnly rack; use NLBInternal instead")
	}

	if nextNLBInternal == "Yes" && nextInternal != "Yes" {
		return fmt.Errorf("cannot enable NLBInternal on a rack without Internal=Yes; set Internal=Yes in the same command")
	}

	if nlbSet && currentNLB == "Yes" && nextNLB == "No" {
		blockers, err := p.appsUsingNLBScheme("public")
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			sort.Strings(blockers)
			return fmt.Errorf("cannot disable NLB: apps %s still declare public nlb ports; remove nlb: from their manifests and redeploy first", strings.Join(blockers, ", "))
		}
		curProt, err := p.stackParameter(p.Rack, "NLBDeletionProtection")
		if err == nil && curProt == "Yes" {
			nextProt, protSet := want("NLBDeletionProtection")
			if !protSet || nextProt == "Yes" {
				return fmt.Errorf("cannot disable NLB while NLBDeletionProtection=Yes; unset protection first, wait for the update to complete, then toggle NLB off")
			}
		}
	}

	if nlbiSet && currentNLBInternal == "Yes" && nextNLBInternal == "No" {
		blockers, err := p.appsUsingNLBScheme("internal")
		if err != nil {
			return err
		}
		if len(blockers) > 0 {
			sort.Strings(blockers)
			return fmt.Errorf("cannot disable NLBInternal: apps %s still declare internal nlb ports; remove nlb: from their manifests and redeploy first", strings.Join(blockers, ", "))
		}
		curProt, err := p.stackParameter(p.Rack, "NLBInternalDeletionProtection")
		if err == nil && curProt == "Yes" {
			nextProt, protSet := want("NLBInternalDeletionProtection")
			if !protSet || nextProt == "Yes" {
				return fmt.Errorf("cannot disable NLBInternal while NLBInternalDeletionProtection=Yes; unset protection first, wait for the update to complete, then toggle NLBInternal off")
			}
		}
	}

	if p.InstanceSecurityGroup != "" {
		if v, ok := want("NLBPreserveClientIP"); ok && v == "Yes" {
			return fmt.Errorf("cannot enable NLBPreserveClientIP on a rack with a customer-supplied InstanceSecurityGroup; your instance SG must add an ingress rule from the NLB security group (exported as ${Rack}:NLBSecurityGroup) for the NLB listener ports before this feature can be enabled safely")
		}
		if v, ok := want("NLBInternalPreserveClientIP"); ok && v == "Yes" {
			return fmt.Errorf("cannot enable NLBInternalPreserveClientIP on a rack with a customer-supplied InstanceSecurityGroup; your instance SG must add an ingress rule from the NLB security group (exported as ${Rack}:NLBInternalSecurityGroup) for the NLB listener ports before this feature can be enabled safely")
		}
	}

	// Inverse interlock: setting a customer InstanceSecurityGroup on a rack
	// where preserve_client_ip is already enabled would break NLB traffic
	// silently (the convox-managed InstancesSecurityNLBIngress no longer
	// applies — the customer SG replaces InstancesSecurity on hosts). Block
	// unless the same call also disables preserve_client_ip.
	if nextSG, sgSet := want("InstanceSecurityGroup"); sgSet && nextSG != "" && p.InstanceSecurityGroup == "" {
		preserveWillBeOn := func(paramName string, curOn bool) bool {
			v, ok := want(paramName)
			if ok {
				return v == "Yes"
			}
			return curOn
		}
		if preserveWillBeOn("NLBPreserveClientIP", p.NLBPreserveClientIP) {
			return fmt.Errorf("cannot set a customer InstanceSecurityGroup while NLBPreserveClientIP=Yes; set NLBPreserveClientIP=No in this same command (or unset it first), then set the custom SG. After the customer SG is in place, re-enable NLBPreserveClientIP only after adding an ingress rule from ${Rack}:NLBSecurityGroup to your SG")
		}
		if preserveWillBeOn("NLBInternalPreserveClientIP", p.NLBInternalPreserveClientIP) {
			return fmt.Errorf("cannot set a customer InstanceSecurityGroup while NLBInternalPreserveClientIP=Yes; set NLBInternalPreserveClientIP=No in this same command (or unset it first), then set the custom SG")
		}
	}

	return nil
}

// validateNLBUninstall blocks `convox rack uninstall` when NLB deletion
// protection is enabled on either NLB. AWS rejects NLB delete calls while
// protection is on, which would strand the rack stack in DELETE_FAILED
// mid-uninstall. Takes the stack name directly so it honors the SystemUninstall
// signature rather than implicitly using p.Rack.
func (p *Provider) validateNLBUninstall(name string) error {
	for _, key := range []string{"NLBDeletionProtection", "NLBInternalDeletionProtection"} {
		v, err := p.stackParameter(name, key)
		if err != nil {
			continue
		}
		if v == "Yes" {
			return fmt.Errorf("cannot uninstall rack while NLB deletion protection is enabled; run 'convox rack params set NLBDeletionProtection=No NLBInternalDeletionProtection=No' first (current: %s=%s)", key, v)
		}
	}
	return nil
}
