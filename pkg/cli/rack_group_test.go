package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveGroup_ExactMatch(t *testing.T) {
	cases := []string{
		"network", "nlb", "security", "scaling", "instances",
		"build", "api", "logging", "storage", "meta",
	}
	for _, g := range cases {
		t.Run(g, func(t *testing.T) {
			got, err := resolveGroup(g)
			require.NoError(t, err)
			require.Equal(t, g, got)
		})
	}
}

func TestResolveGroup_CaseInsensitiveAndTrim(t *testing.T) {
	cases := map[string]string{
		"NETWORK":     "network",
		"Network":     "network",
		"  network  ": "network",
		"\tNLB\n":     "nlb",
		" SECURITY ":  "security",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := resolveGroup(input)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestResolveGroup_UniquePrefix(t *testing.T) {
	cases := map[string]string{
		"net":  "network",
		"netw": "network",
		"nl":   "nlb",
		"sec":  "security",
		"sto":  "storage",
		"sca":  "scaling",
		"in":   "instances",
		"ins":  "instances",
		"bu":   "build",
		"log":  "logging",
		"me":   "meta",
		"ap":   "api",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := resolveGroup(input)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestResolveGroup_AmbiguousPrefix(t *testing.T) {
	_, err := resolveGroup("n")
	require.Error(t, err)
	msg := err.Error()
	require.Contains(t, msg, "matches multiple groups")
	require.Contains(t, msg, "network")
	require.Contains(t, msg, "nlb")
}

func TestResolveGroup_Unknown(t *testing.T) {
	_, err := resolveGroup("not-a-group")
	require.Error(t, err)
	msg := err.Error()
	require.Contains(t, msg, "not found")
	require.Contains(t, msg, "available groups")
	require.Contains(t, msg, "network")
}

func TestResolveGroup_Empty(t *testing.T) {
	_, err := resolveGroup("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "group name required")
}

func TestResolveGroup_WhitespaceOnly(t *testing.T) {
	_, err := resolveGroup("   ")
	require.Error(t, err)
	require.Contains(t, err.Error(), "group name required")
}

func TestResolveGroup_AmbiguityHintFormat(t *testing.T) {
	_, err := resolveGroup("n")
	require.Error(t, err)
	msg := err.Error()
	// Hint uses disambiguatingPrefix (3-char min). For 'n' -> 'network', 'nlb':
	// disambiguatingPrefix("network") = "net"; disambiguatingPrefix("nlb") = "nlb" (already 3 chars).
	// Resulting hint: "(use 'net' or 'nlb')"
	require.Contains(t, msg, "(use 'net' or 'nlb')")
}

func TestResolveGroup_FormatGroupListShowsAllTen(t *testing.T) {
	_, err := resolveGroup("invalid-name")
	require.Error(t, err, "expected error for unknown group")
	msg := err.Error()
	for _, g := range []string{
		"api", "build", "instances", "logging", "meta",
		"network", "nlb", "scaling", "security", "storage",
	} {
		require.Contains(t, msg, g, "expected group %q in error output", g)
	}
}

func TestDisambiguatingPrefix_ThreeCharMin(t *testing.T) {
	cases := map[string]string{
		"network":   "net",
		"nlb":       "nlb",
		"security":  "sec",
		"storage":   "sto",
		"scaling":   "sca",
		"instances": "ins",
		"build":     "bui",
		"logging":   "log",
		"meta":      "met",
		"api":       "api",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			got := disambiguatingPrefix(input)
			require.Equal(t, want, got, "disambiguating prefix for %q", input)
		})
	}
}

// TestFormatGroupList_SortedWithDescriptions asserts the group list in error
// output is sorted and each group is followed by its description. The expected
// prefix is "\n  "+group+" " (2 leading spaces) per V3 parity — see
// formatGroupList in rack.go.
func TestFormatGroupList_SortedWithDescriptions(t *testing.T) {
	_, err := resolveGroup("trigger-the-error-path")
	require.Error(t, err)
	msg := err.Error()

	expected := []string{"api", "build", "instances", "logging", "meta", "network", "nlb", "scaling", "security", "storage"}
	lastIdx := -1
	for _, g := range expected {
		idx := strings.Index(msg, "\n  "+g+" ")
		require.GreaterOrEqual(t, idx, 0, "expected group %q to appear in formatted error output (2-space-indented)", g)
		require.Greater(t, idx, lastIdx, "group %q should appear in sorted order", g)
		lastIdx = idx
	}
}

// TestSensitiveParamsList asserts the exact mask list. Guards against
// accidental additions (non-credential params slipping into masking) or
// removals (credentials slipping OUT of masking).
func TestSensitiveParamsList(t *testing.T) {
	want := map[string]bool{
		"Password":  true,
		"HttpProxy": true,
	}
	require.Equal(t, want, sensitiveParams, "sensitiveParams must equal the exact mask list per spec")

	for _, k := range []string{
		"Key",
		"SyslogDestination",
		"InstancePolicy",
		"BuildInstancePolicy",
		"LogBucket",
		"ClientId",
		"WhiteList",
		"NLBAllowCIDR",
		"Encryption",
		"VPCCIDR",
		"Ami",
		"Version",
	} {
		require.False(t, sensitiveParams[k], "%s must NOT be in sensitiveParams per spec rubric", k)
	}
}

// TestParamGroupsCoverRackJSON asserts that every Parameter in
// provider/aws/formation/rack.json appears in at least one group, and
// every group member appears in rack.json. Dual-listing IS allowed — a
// param may appear in multiple groups.
func TestParamGroupsCoverRackJSON(t *testing.T) {
	data, err := os.ReadFile("../../provider/aws/formation/rack.json")
	require.NoError(t, err, "must be able to read rack.json from test cwd")

	var rack struct {
		Parameters map[string]json.RawMessage `json:"Parameters"`
	}
	require.NoError(t, json.Unmarshal(data, &rack))
	require.NotEmpty(t, rack.Parameters, "rack.json should have Parameters")

	allGroupMembers := map[string]bool{}
	for _, gmembers := range paramGroups {
		for k := range gmembers {
			allGroupMembers[k] = true
		}
	}

	var orphans []string
	for k := range rack.Parameters {
		if !allGroupMembers[k] {
			orphans = append(orphans, k)
		}
	}
	require.Empty(t, orphans, "rack.json params missing from any group in paramGroups: %v", orphans)

	var stale []string
	for k := range allGroupMembers {
		if _, ok := rack.Parameters[k]; !ok {
			stale = append(stale, k)
		}
	}
	require.Empty(t, stale, "paramGroups members not in rack.json Parameters: %v", stale)

	require.Equal(t, 110, len(rack.Parameters), "post-hardening rack.json should have 110 Parameters")
}
