package aws

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
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
// declares at least one service with an NLB port of the given scheme ("public" or "internal"),
// counting only ports that set preserve_client_ip: true when preserveOnly is set.
// Returns entries formatted as "app/service".
//
// Fails open on per-app errors (logs a warning and continues) so a single broken release does
// not block rack-wide operations.
func (p *Provider) appsUsingNLBScheme(scheme string, preserveOnly bool) ([]string, error) {
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
				if np.Scheme == scheme && (!preserveOnly || (np.PreserveClientIP != nil && *np.PreserveClientIP)) {
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
//   - Custom InstanceSecurityGroup must allow all traffic from the NLB SG while preserve_client_ip is in use
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
		blockers, err := p.appsUsingNLBScheme("public", false)
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
		blockers, err := p.appsUsingNLBScheme("internal", false)
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

	nextSG, sgSet := want("InstanceSecurityGroup")
	if !sgSet {
		nextSG = p.InstanceSecurityGroup
	}
	if nextSG == "" {
		return nil
	}
	sgChanged := nextSG != p.InstanceSecurityGroup

	for _, s := range []struct {
		scheme, nlbParam, preserveParam string
		nlbOn, nlbNext, preserveOn      bool
	}{
		{"public", "NLB", "NLBPreserveClientIP", p.NLB, nextNLB == "Yes", p.NLBPreserveClientIP},
		{"internal", "NLBInternal", "NLBInternalPreserveClientIP", p.NLBInternal, nextNLBInternal == "Yes", p.NLBInternalPreserveClientIP},
	} {
		if !s.nlbNext {
			continue
		}
		v, set := want(s.preserveParam)
		preserveNext := s.preserveOn
		if set {
			preserveNext = v == "Yes"
		}
		if !s.nlbOn {
			if preserveNext {
				return fmt.Errorf("cannot enable %s while %s=Yes on a rack with a custom InstanceSecurityGroup; set %s=No in this command, add an ingress rule to %s allowing all traffic from the new %s:%sSecurityGroup, then set %s=Yes", s.nlbParam, s.preserveParam, s.preserveParam, nextSG, p.Rack, s.nlbParam, s.preserveParam)
			}
			continue
		}
		check := set && v == "Yes"
		if !check && sgChanged {
			check = s.preserveOn
			if !check {
				apps, err := p.appsUsingNLBScheme(s.scheme, true)
				if err != nil {
					return err
				}
				check = len(apps) > 0
			}
		}
		if check {
			if err := p.checkNLBIngressRule(nextSG, s.scheme); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkNLBIngressRule returns an error unless sg allows all traffic from the scheme's NLB security group.
func (p *Provider) checkNLBIngressRule(sg, scheme string) error {
	output := "NLBSecurityGroup"
	if scheme == "internal" {
		output = "NLBInternalSecurityGroup"
	}

	s, err := p.describeStack(p.Rack)
	if err != nil {
		return fmt.Errorf("could not read rack stack %s: %s", p.Rack, err)
	}

	nlbSG := stackOutputs(s)[output]
	if nlbSG == "" {
		return fmt.Errorf("rack stack has no %s output", output)
	}

	res, err := p.ec2().DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{aws.String(sg)},
	})
	if err != nil {
		return fmt.Errorf("could not read InstanceSecurityGroup %s: %s", sg, err)
	}

	for _, g := range res.SecurityGroups {
		for _, perm := range g.IpPermissions {
			if aws.StringValue(perm.IpProtocol) != "-1" {
				continue
			}
			for _, pair := range perm.UserIdGroupPairs {
				if aws.StringValue(pair.GroupId) == nlbSG {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("preserve client IP on the %s NLB needs an ingress rule on InstanceSecurityGroup %s allowing all traffic from the NLB security group %s (%s:%s); add that rule and retry", scheme, sg, nlbSG, p.Rack, output)
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
