package helpers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/structs"
)

func EnvDiff(a, b string) (string, error) {
	ae := structs.Environment{}
	be := structs.Environment{}

	if err := ae.Load([]byte(strings.TrimSpace(a))); err != nil {
		return "", err
	}

	if err := be.Load([]byte(strings.TrimSpace(b))); err != nil {
		return "", err
	}

	adds := []string{}
	changes := []string{}
	removes := []string{}

	for k := range be {
		if v, ok := ae[k]; ok {
			if be[k] != v {
				changes = append(changes, k)
			}
		} else {
			adds = append(adds, k)
		}
	}

	for k := range ae {
		if _, ok := be[k]; !ok {
			removes = append(removes, k)
		}
	}

	sort.Strings(adds)
	sort.Strings(changes)
	sort.Strings(removes)

	desc := ""

	for _, k := range adds {
		desc = fmt.Sprintf("%s add:%s", desc, k)
	}

	for _, k := range changes {
		desc = fmt.Sprintf("%s change:%s", desc, k)
	}

	for _, k := range removes {
		desc = fmt.Sprintf("%s remove:%s", desc, k)
	}

	return strings.TrimSpace(desc), nil
}
