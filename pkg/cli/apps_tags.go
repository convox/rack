package cli

import "strings"

const v2TagsFloor = "20260929232050"

func parseTags(s string) map[string]string {
	tags := map[string]string{}

	for _, seg := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(seg, "=")
		if !ok {
			continue
		}

		if _, has := tags[k]; !has {
			tags[k] = v
		}
	}

	return tags
}

func tagsApplied(input, readback, rackTags, rackVersion string) bool {
	in := parseTags(input)

	if len(in) == 0 {
		return readback == input
	}

	got := parseTags(readback)
	rk := map[string]string{}

	if rackVersion > v2TagsFloor {
		rk = parseTags(rackTags)

		for _, r := range []string{"App", "System", "Type", "Version", "Generation", "Name", "Rack"} {
			delete(rk, r)
		}
	}

	for k, v := range in {
		if gv, ok := got[k]; ok {
			if gv != v {
				return false
			}
			continue
		}

		if rv, ok := rk[k]; !ok || rv != v {
			return false
		}
	}

	return true
}
