package cli

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"

	pv "github.com/convox/rack/provider"
	cv "github.com/convox/version"
)

// sensitiveParams enumerates V2 rack params whose values render as
// "**********" on a TTY when --reveal is not passed. Pipe output and
// --reveal bypass masking. Empty values are never masked.
var sensitiveParams = map[string]bool{
	"Password":  true,
	"HttpProxy": true,
}

// paramGroups categorizes V2 rack params into curated logical groups for the
// `convox rack params -g <group>` filter. A param may belong to multiple
// groups (5 V2 params are dual-listed). Every V2 rack.json Parameter must
// land in at least one group — enforced by TestParamGroupsCoverRackJSON.
//
// When adding a new rack param to provider/aws/formation/rack.json, add it
// to the appropriate group(s) below. The drift-detector test will fail
// otherwise.
var paramGroups = map[string]map[string]bool{
	"network": {
		"AvailabilityZones":    true,
		"ExistingVpc":          true,
		"HttpProxy":            true,
		"Internal":             true,
		"InternalOnly":         true,
		"InternetGateway":      true,
		"MaxAvailabilityZones": true,
		"PlaceLambdaInVpc":     true,
		"Private":              true,
		"Subnet0CIDR":          true,
		"Subnet1CIDR":          true,
		"Subnet2CIDR":          true,
		"SubnetPrivate0CIDR":   true,
		"SubnetPrivate1CIDR":   true,
		"SubnetPrivate2CIDR":   true,
		"VPCCIDR":              true,
	},
	"nlb": {
		"NLB":                           true,
		"NLBAllowCIDR":                  true,
		"NLBCrossZone":                  true,
		"NLBDeletionProtection":         true,
		"NLBInternal":                   true,
		"NLBInternalAllowCIDR":          true,
		"NLBInternalCrossZone":          true,
		"NLBInternalDeletionProtection": true,
		"NLBInternalPreserveClientIP":   true,
		"NLBPreserveClientIP":           true,
	},
	"security": {
		"BuildInstancePolicy":                   true, // dual-listed in build
		"BuildInstanceSecurityGroup":            true,
		"EnableContainerReadonlyRootFilesystem": true,
		"EnableSharedEFSVolumeEncryption":       true, // dual-listed in storage
		"EncryptEbs":                            true, // dual-listed in storage
		"Encryption":                            true,
		"HttpProxy":                             true, // dual-listed in network
		"IMDSHttpPutResponseHopLimit":           true,
		"IMDSHttpTokens":                        true,
		"InstancePolicy":                        true, // dual-listed in instances
		"InstanceSecurityGroup":                 true,
		"InstancesIpToIncludInWhiteListing":     true,
		"Key":                                   true,
		"Password":                              true,
		"PrivateApiSecurityGroup":               true,
		"RouterInternalSecurityGroup":           true,
		"RouterSecurityGroup":                   true,
		"SslPolicy":                             true,
		"WhiteList":                             true,
	},
	"scaling": {
		"Autoscale":                      true,
		"AutoscaleExtra":                 true,
		"HighAvailability":               true,
		"InstanceCount":                  true,
		"InstanceUpdateBatchSize":        true,
		"NoHAAutoscaleExtra":             true,
		"NoHaInstanceCount":              true,
		"OnDemandMinCount":               true,
		"ScheduleRackScaleDown":          true,
		"ScheduleRackScaleUp":            true,
		"SpotFleetAllocationStrategy":    true,
		"SpotFleetAllowedInstanceTypes":  true,
		"SpotFleetExcludedInstanceTypes": true,
		"SpotFleetMaxPrice":              true,
		"SpotFleetMinMemoryMiB":          true,
		"SpotFleetMinOnDemandCount":      true,
		"SpotFleetMinVcpuCount":          true,
		"SpotFleetTargetType":            true,
		"SpotInstanceBid":                true,
	},
	"instances": {
		"Ami":                 true,
		"CpuCredits":          true,
		"DefaultAmi":          true,
		"DefaultAmiArm":       true,
		"InstanceBootCommand": true,
		"InstancePolicy":      true, // dual-listed in security
		"InstanceRunCommand":  true,
		"InstanceType":        true,
		"SwapSize":            true,
		"Tenancy":             true,
		"VolumeSize":          true,
	},
	"build": {
		"BuildCpu":                    true,
		"BuildImage":                  true,
		"BuildInstance":               true,
		"BuildInstancePolicy":         true, // dual-listed in security
		"BuildMemory":                 true,
		"BuildMethod":                 true,
		"BuildVolumeSize":             true,
		"FargateBuildCpu":             true,
		"FargateBuildMemory":          true,
		"PrivateBuild":                true,
		"PruneOlderImagesCronRunFreq": true,
		"PruneOlderImagesInHour":      true,
	},
	"api": {
		"ApiCount":                true,
		"ApiCpu":                  true,
		"ApiMonitorMemory":        true,
		"ApiRouter":               true,
		"ApiWebMemory":            true,
		"DisableALBPort80":        true,
		"InternalRouterSuffix":    true,
		"LoadBalancerIdleTimeout": true,
		"PrivateApi":              true,
		"RouterMitigationMode":    true,
	},
	"logging": {
		"LogBucket":         true,
		"LogDriver":         true,
		"LogRetention":      true,
		"SyslogDestination": true,
		"SyslogFormat":      true,
	},
	"storage": {
		"DynamoDbTableDeletionProtectionEnabled":  true,
		"DynamoDbTablePointInTimeRecoveryEnabled": true,
		"EnableS3Versioning":                      true,
		"EnableSharedEFSVolumeEncryption":         true, // dual-listed in security
		"EncryptEbs":                              true, // dual-listed in security
	},
	"meta": {
		"ClientId":                true,
		"Development":             true,
		"EcsContainerStopTimeout": true,
		"EcsPollInterval":         true,
		"ImagePullBehavior":       true,
		"MaintainTimerState":      true,
		"Telemetry":               true,
		"Version":                 true,
	},
}

// groupDescriptions provides the one-line label shown next to each group
// name in error output (e.g., unknown-group and ambiguous-prefix errors).
// Keep in sync with paramGroups keys.
var groupDescriptions = map[string]string{
	"network":   "VPC, subnets, gateways, connectivity, proxy",
	"nlb":       "Network Load Balancer: listeners, cross-zone, allow-CIDR, preserve-client-IP",
	"security":  "Credentials, allowlists, SGs, SSL, IMDS, container hardening",
	"scaling":   "Autoscaling, spot fleet, instance counts, schedules, HA",
	"instances": "AMI, instance type, boot/run commands, IAM policy, volumes, tenancy",
	"build":     "Build method, build instance, Fargate build, image pruning",
	"api":       "Rack API web process config, router, ingress toggles",
	"logging":   "Logs destination, retention, syslog format",
	"storage":   "S3 versioning, DynamoDB protection, EFS encryption",
	"meta":      "Version, development mode, telemetry, client ID, ECS tuning",
}

// resolveGroup resolves a possibly-partial group name to an exact group key.
// Priority: exact match > unique prefix match. Case-insensitive. Whitespace
// is trimmed. Returns an error listing candidates or all groups on
// ambiguous / unknown / empty input.
func resolveGroup(input string) (string, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "", fmt.Errorf("group name required\n  %s", formatGroupList())
	}

	if _, ok := paramGroups[input]; ok {
		return input, nil
	}

	var matches []string
	for g := range paramGroups {
		if strings.HasPrefix(g, input) {
			matches = append(matches, g)
		}
	}
	sort.Strings(matches)

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("group '%s' not found\n  %s", input, formatGroupList())
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("group '%s' matches multiple groups: %s %s\n  %s",
			input, strings.Join(matches, ", "), formatAmbiguousHint(matches), formatGroupList())
	}
}

// formatGroupList returns a sorted, padded two-column listing of all
// available groups for inclusion in error output.
func formatGroupList() string {
	names := make([]string, 0, len(groupDescriptions))
	maxLen := 0
	for g := range groupDescriptions {
		names = append(names, g)
		if len(g) > maxLen {
			maxLen = len(g)
		}
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("available groups:\n")
	for _, g := range names {
		b.WriteString(fmt.Sprintf("  %-*s    %s\n", maxLen, g, groupDescriptions[g]))
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatAmbiguousHint returns a parenthesized hint showing a short-but-
// readable disambiguating prefix for each ambiguous candidate, e.g.,
// "(use 'net' or 'nlb')" for candidates ["network", "nlb"].
func formatAmbiguousHint(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	hints := make([]string, 0, len(candidates))
	for _, c := range candidates {
		hints = append(hints, "'"+disambiguatingPrefix(c)+"'")
	}
	switch len(hints) {
	case 1:
		return "(use " + hints[0] + ")"
	case 2:
		return "(use " + hints[0] + " or " + hints[1] + ")"
	default:
		return "(use " + strings.Join(hints[:len(hints)-1], ", ") + ", or " + hints[len(hints)-1] + ")"
	}
}

// disambiguatingPrefix returns a short-but-readable prefix of `group` that
// resolves uniquely against all paramGroups keys. Uses a 3-character
// minimum for human readability — a technically-unique 1- or 2-char
// prefix like "ne" for "network" reads like an abbreviation and is avoided.
func disambiguatingPrefix(group string) string {
	const minLen = 3
	if len(group) <= minLen {
		return group
	}
	for n := minLen; n <= len(group); n++ {
		prefix := group[:n]
		hits := 0
		for g := range paramGroups {
			if strings.HasPrefix(g, prefix) {
				hits++
				if hits > 1 {
					break
				}
			}
		}
		if hits == 1 {
			return prefix
		}
	}
	return group
}

func init() {
	register("rack", "get information about the rack", Rack, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("rack access", "get rack access creds", RackAccess, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.StringFlag("role", "", "access role: read or write"),
			stdcli.IntFlag("duration-in-hours", "", "duration in hours"),
		},
		Validate: stdcli.Args(0),
	})

	register("rack access key rotate", "rotate access key", RackAccessKeyRotate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	registerWithoutProvider("rack install", "install a rack", RackInstall, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemInstallOptions{})),
		Usage:    "<type> [Parameter=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	register("rack logs", "get logs for the rack", RackLogs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.LogsOptions{}), flagNoFollow, flagRack),
		Validate: stdcli.Args(0),
	})

	register("rack params", "display rack parameters", RackParams, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.StringFlag("group", "g", "filter to a param group (invalid name lists all)"),
			stdcli.BoolFlag("reveal", "", "show unmasked param values"),
		},
		Validate: stdcli.Args(0),
	})

	register("rack params set", "set rack parameters", RackParamsSet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<Key=Value> [Key=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	register("rack ps", "list rack processes", RackPs, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemProcessesOptions{}), flagRack),
		Validate: stdcli.Args(0),
	})

	register("rack releases", "list rack version history", RackReleases, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("rack runtimes", "list of attachable runtime integrations", RackRuntimes, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("rack runtime attach", "attach runtime integration", RackRuntimeAttach, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(1),
	})

	register("rack scale", "scale the rack", RackScale, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.IntFlag("count", "c", "instance count"),
			stdcli.StringFlag("type", "t", "instance type"),
		},
		Validate: stdcli.Args(0),
	})

	registerWithoutProvider("rack uninstall", "uninstall a rack", RackUninstall, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.SystemUninstallOptions{})),
		Usage:    "<type> <name>",
		Validate: stdcli.Args(2),
	})

	register("rack sync", "sync v2 rack API url", RackSync, stdcli.CommandOptions{
		Flags: []stdcli.Flag{flagRack, stdcli.StringFlag("name", "n", "passing the rack name it will display the Rack API URL")},
	})

	register("rack sync whitelist instances ip", "sync  whitelist instances ips in security group", RackSyncWhiteListInstancesIp, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("rack update", "update the rack", RackUpdate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Validate: stdcli.ArgsMax(1),
	})

	register("rack wait", "wait for rack to finish updating", RackWait, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})
}

func Rack(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Name", s.Name)
	i.Add("Provider", s.Provider)

	if s.Region != "" {
		i.Add("Region", s.Region)
	}

	if s.Domain != "" {
		if ri := s.Outputs["DomainInternal"]; ri != "" {
			i.Add("Router", fmt.Sprintf("%s (external)\n%s (internal)", s.Domain, ri))
		} else {
			i.Add("Router", s.Domain)
		}
	}

	if nlb := s.Outputs["NLBHost"]; nlb != "" {
		var eips []string
		for _, k := range []string{"NLBEIP0", "NLBEIP1", "NLBEIP2"} {
			if v := s.Outputs[k]; v != "" {
				eips = append(eips, v)
			}
		}
		if len(eips) > 0 {
			i.Add("NLB", fmt.Sprintf("%s (%s)", nlb, strings.Join(eips, ", ")))
		} else {
			i.Add("NLB", nlb)
		}
	}
	if nlbi := s.Outputs["NLBInternalHost"]; nlbi != "" {
		i.Add("NLB Internal", nlbi)
	}

	i.Add("Status", s.Status)
	i.Add("Version", s.Version)

	return i.Print()
}

func RackAccess(rack sdk.Interface, c *stdcli.Context) error {
	rData, err := rack.SystemGet()
	if err != nil {
		return err
	}

	role, ok := c.Value("role").(string)
	if !ok {
		return fmt.Errorf("role is required")
	}

	duration, ok := c.Value("duration-in-hours").(int)
	if !ok {
		return fmt.Errorf("duration is required")
	}

	jwtTk, err := rack.SystemJwtToken(structs.SystemJwtOptions{
		Role:           options.String(role),
		DurationInHour: options.String(strconv.Itoa(duration)),
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.Writef("RACK_URL=https://jwt:%s@%s\n", jwtTk.Token, rData.RackDomain)
}

func RackAccessKeyRotate(rack sdk.Interface, c *stdcli.Context) error {
	_, err := rack.SystemJwtSignKeyRotate()
	if err != nil {
		return err
	}

	return c.OK()
}

func RackInstall(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.SystemInstallOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if opts.Version == nil {
		v, err := cv.Latest()
		if err != nil {
			return err
		}
		opts.Version = options.String(v)
	}

	if id, _ := c.SettingRead("id"); id != "" {
		opts.Id = options.String(id)
	}

	opts.Parameters = map[string]string{}

	for _, arg := range c.Args[1:] {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return fmt.Errorf("Key=Value expected: %s", arg)
		}

		opts.Parameters[parts[0]] = parts[1]
	}

	if err := validateParams(opts.Parameters); err != nil {
		return err
	}

	p, err := pv.FromName(c.Arg(0))
	if err != nil {
		return err
	}

	// if !helpers.DefaultBool(opts.Raw, false) {
	//   c.Writef("   ___ ___  _  _ _   __ __ _  __\n")
	//   c.Writef("  / __/ _ \\| \\| \\ \\ / / _ \\ \\/ /\n")
	//   c.Writef(" | (_| (_) |  ` |\\ V / (_) )  ( \n")
	//   c.Writef("  \\___\\___/|_|\\_| \\_/ \\___/_/\\_\\\n")
	//   c.Writef("\n")
	// }

	ep, err := p.SystemInstall(c, opts)
	if err != nil {
		return err
	}

	u, err := url.Parse(ep)
	if err != nil {
		return err
	}

	password := ""

	if u.User != nil {
		if pw, ok := u.User.Password(); ok {
			password = pw
		}
	}

	if err := c.SettingWriteKey("auth", u.Host, password); err != nil {
		return err
	}

	if err := c.SettingWriteKey("self-managed", *opts.Name, u.Host); err != nil {
		return err
	}

	if host, _ := c.SettingRead("host"); host == "" {
		c.SettingWrite("host", u.Host)
	}

	return nil
}

func RackLogs(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	if c.Bool("no-follow") {
		opts.Follow = options.Bool(false)
	}

	opts.Prefix = options.Bool(true)

	r, err := rack.SystemLogs(opts)
	if err != nil {
		return err
	}

	io.Copy(c, r)

	return nil
}

func RackParams(rack sdk.Interface, c *stdcli.Context) error {
	var (
		groupFilter   map[string]bool
		resolvedGroup string
	)
	if groupInput := c.String("group"); groupInput != "" {
		resolved, rerr := resolveGroup(groupInput)
		if rerr != nil {
			return rerr
		}
		resolvedGroup = resolved
		groupFilter = paramGroups[resolved]
	}

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	keys := []string{}
	for k := range s.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	shouldMask := !c.Bool("reveal") && IsTerminalFn(c)

	i := c.Info()
	rowsAdded := 0

	for _, k := range keys {
		if groupFilter != nil && !groupFilter[k] {
			continue
		}
		v := s.Parameters[k]
		if shouldMask && sensitiveParams[k] && v != "" {
			v = "**********"
		}
		i.Add(k, v)
		rowsAdded++
	}

	if err := i.Print(); err != nil {
		return err
	}

	if groupFilter != nil && rowsAdded == 0 {
		// Write via stdcli's captured writer so test harnesses observe the
		// NOTICE. V3 writes to os.Stderr directly; V2 diverges here because
		// V2's test infrastructure routes stderr through the Writer.
		fmt.Fprintf(c.Writer().Stderr, "NOTICE: no params in group '%s' for this rack\n", resolvedGroup)
	}

	return nil
}

func RackParamsSet(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	opts := structs.SystemUpdateOptions{
		Parameters: map[string]string{},
	}

	for _, arg := range c.Args {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return fmt.Errorf("Key=Value expected: %s", arg)
		}

		if parts[0] == "HighAvailability" {
			return errors.New("the HighAvailability parameter is only supported during rack installation")
		}

		opts.Parameters[parts[0]] = parts[1]
	}

	if err := validateParams(opts.Parameters); err != nil {
		return err
	}

	c.Startf("Updating parameters")

	if s.Version <= "20180708231844" {
		if err := rack.AppParametersSet(s.Name, opts.Parameters); err != nil {
			return err
		}
	} else {
		if err := rack.SystemUpdate(opts); err != nil {
			return err
		}
	}

	// If NLB/NLBInternal was just enabled, hint about provisioning time.
	for _, k := range []string{"NLB", "NLBInternal"} {
		if next, ok := opts.Parameters[k]; ok && next == "Yes" && s.Parameters[k] != "Yes" {
			c.Writef("NLB provisioning typically takes 5-10 minutes; check status with 'convox rack'.\n")
			break
		}
	}

	if c.Bool("wait") {
		c.Writef("\n")

		if err := helpers.WaitForRackWithLogs(rack, c); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackPs(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.SystemProcessesOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	ps, err := rack.SystemProcesses(opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "APP", "SERVICE", "STATUS", "RELEASE", "STARTED", "COMMAND")

	for _, p := range ps {
		t.AddRow(p.Id, p.App, p.Name, p.Status, p.Release, helpers.Ago(p.Started), p.Command)
	}

	return t.Print()
}

func RackReleases(rack sdk.Interface, c *stdcli.Context) error {
	rs, err := rack.SystemReleases()
	if err != nil {
		return err
	}

	t := c.Table("VERSION", "UPDATED")

	for _, r := range rs {
		t.AddRow(r.Id, helpers.Ago(r.Created))
	}

	return t.Print()
}

func RackRuntimes(rack sdk.Interface, c *stdcli.Context) error {
	host, err := currentHost(c)
	if err != nil {
		c.Fail(err)
	}

	rname := currentRack(c, host)

	endpoint, err := currentEndpoint(c, "")
	if err != nil {
		return err
	}

	p, err := sdk.New(endpoint)
	if err != nil {
		return err
	}

	p.Authenticator = authenticator(c)
	p.Session = currentSession(c)

	rs, err := p.Runtimes(rname)
	if err != nil {
		return err
	}

	t := c.Table("ID", "TITLE")
	for _, r := range rs {
		t.AddRow(r.Id, r.Title)
	}

	return t.Print()
}

func RackRuntimeAttach(rack sdk.Interface, c *stdcli.Context) error {
	host, err := currentHost(c)
	if err != nil {
		c.Fail(err)
	}

	rname := currentRack(c, host)

	endpoint, err := currentEndpoint(c, "")
	if err != nil {
		return err
	}

	p, err := sdk.New(endpoint)
	if err != nil {
		return err
	}

	p.Authenticator = authenticator(c)
	p.Session = currentSession(c)

	if err := p.RuntimeAttach(rname, structs.RuntimeAttachOptions{
		Runtime: &c.Args[0],
	}); err != nil {
		return err
	}

	return c.OK()
}

func RackScale(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var opts structs.SystemUpdateOptions
	update := false

	if v, ok := c.Value("count").(int); ok {
		opts.Count = options.Int(v)
		update = true
	}

	if v, ok := c.Value("type").(string); ok {
		opts.Type = options.String(v)
		update = true
	}

	if update {
		c.Startf("Scaling rack")

		if err := rack.SystemUpdate(opts); err != nil {
			return err
		}

		return c.OK()
	}

	i := c.Info()

	i.Add("Autoscale", s.Parameters["Autoscale"])
	i.Add("Count", fmt.Sprintf("%d", s.Count))
	i.Add("Status", s.Status)
	i.Add("Type", s.Type)

	return i.Print()
}

func RackSync(rack sdk.Interface, c *stdcli.Context) error {
	if c.String("name") != "" {
		c.Startf("Fetching rack API URL...")
		c.Writef("\n")

		if os.Getenv("AWS_REGION") != "" && os.Getenv("AWS_DEFAULT_REGION") != "" {
			return c.Errorf("region is not defined, please set AWS_DEFAULT_REGION")
		}

		rname := c.String("name")
		// deleting the org name
		parts := strings.Split(rname, "/")
		if len(parts) == 2 {
			rname = parts[1]
		}

		s, err := helpers.NewSession()
		if err != nil {
			return err
		}

		cf := cloudformation.New(s)
		o, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(rname)})
		if err != nil {
			return err
		}

		if len(o.Stacks) == 0 {
			return c.Errorf("formation stack with name %s not found", rname)
		}

		st := o.Stacks[0]
		for _, o := range st.Outputs {
			if *o.OutputKey == "Dashboard" {
				c.Writef("url=%s\n", *o.OutputValue)
			}
		}

		return c.OK()
	}

	c.Startf("Synchronizing rack API URL...")
	c.Writef("\n")

	host, err := currentHost(c)
	if err != nil {
		c.Fail(err)
	}
	rname := currentRack(c, host)

	err = rack.Sync(rname)
	if err != nil {
		return err
	}

	return c.OK()
}

func RackUninstall(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.SystemUninstallOptions
	name := c.Arg(1)

	if err := c.Options(&opts); err != nil {
		return err
	}

	if c.Reader().IsTerminal() {
		opts.Input = c.Reader()
	} else {
		if !c.Bool("force") {
			return fmt.Errorf("must use --force for non-interactive uninstall")
		}
	}

	p, err := pv.FromName(c.Arg(0))
	if err != nil {
		return err
	}

	c.SettingDeleteKey("self-managed", name)

	if err := p.SystemUninstall(name, c, opts); err != nil {
		return err
	}

	return nil
}

func RackUpdate(rack sdk.Interface, c *stdcli.Context) error {
	target := c.Arg(0)

	// if no version specified, find the next version
	if target == "" {
		s, err := rack.SystemGet()
		if err != nil {
			return err
		}

		if s.Version == "dev" {
			target = "dev"
		} else {
			v, err := cv.Next(s.Version)
			if err != nil {
				return err
			}

			target = v
		}
	}

	c.Startf("Updating to <release>%s</release>", target)

	if err := rack.SystemUpdate(structs.SystemUpdateOptions{Version: options.String(target)}); err != nil {
		return err
	}

	if c.Bool("wait") {
		c.Writef("\n")

		if err := helpers.WaitForRackWithLogs(rack, c); err != nil {
			return err
		}
	}

	return c.OK()
}

func RackWait(rack sdk.Interface, c *stdcli.Context) error {
	c.Startf("Waiting for rack")

	c.Writef("\n")

	if err := helpers.WaitForRackWithLogs(rack, c); err != nil {
		return err
	}

	return c.OK()
}

func RackSyncWhiteListInstancesIp(rack sdk.Interface, c *stdcli.Context) error {
	err := rack.SyncInstancesIpInSecurityGroup()
	if err != nil {
		return err
	}

	return c.OK()
}

// validateParams validate parameters for install and update rack
func validateParams(params map[string]string) error {
	srdown, srup := params["ScheduleRackScaleDown"], params["ScheduleRackScaleUp"]
	if (srdown == "" || srup == "") && (srdown != "" || srup != "") {
		return fmt.Errorf("to configure ScheduleAction you need both ScheduleRackScaleDown and ScheduleRackScaleUp parameters")
	}

	if params["LogDriver"] == "Syslog" && params["SyslogDestination"] == "" {
		return fmt.Errorf("to enable Syslog as the LogDriver you must pass SyslogDestination parameter")
	}

	return nil
}
