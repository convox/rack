package aws

import (
	"fmt"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/structs"
)

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

// validateNLBParams enforces the NLB rack-param rules:
//   - NLB=Yes incompatible with InternalOnly=Yes
//   - NLBInternal=Yes requires Internal=Yes (either already set or set in the same call)
//   - Disabling NLB/NLBInternal requires no dependent gen2 apps
func (p *Provider) validateNLBParams(opts structs.SystemUpdateOptions) error {
	want := func(key string) (string, bool) {
		if opts.Parameters == nil {
			return "", false
		}
		v, ok := opts.Parameters[key]
		return v, ok
	}

	yesNo := func(b bool) string {
		if b {
			return "Yes"
		}
		return "No"
	}

	currentNLB := yesNo(p.NLB)
	currentNLBInternal := yesNo(p.NLBInternal)
	currentInternal := yesNo(p.Internal)
	currentInternalOnly := yesNo(p.InternalOnly)

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
	}

	return nil
}
