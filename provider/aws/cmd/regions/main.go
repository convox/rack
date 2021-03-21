package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/pkg/errors"
)

type Region struct {
	AvailabilityZones []string
	EFS               bool
	ELBAccountId      string
	Fargate           bool
}

type Regions map[string]Region

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %+v\n", err)
	}
}

func run() error {
	regions, err := fetchRegions()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := fetchAvailabilityZones(regions); err != nil {
		return errors.WithStack(err)
	}

	if err := fetchEFS(regions); err != nil {
		return errors.WithStack(err)
	}

	if err := fetchFargate(regions); err != nil {
		return errors.WithStack(err)
	}

	if err := fetchELBAccountIds(regions); err != nil {
		return errors.WithStack(err)
	}

	names := []string{}

	for name := range regions {
		names = append(names, name)
	}

	sort.Strings(names)

	rns := make([]string, len(names))
	efss := make([]string, len(names))
	tazs := make([]string, len(names))
	elbs := make([]string, len(names))
	fargates := make([]string, len(names))

	for i, name := range names {
		region := regions[name]

		rns[i] = fmt.Sprintf("%q:", name)
		efss[i] = fmt.Sprintf(`"EFS": %q,`, yn(region.EFS))
		tazs[i] = fmt.Sprintf(`"ThirdAvailabilityZone": %q,`, yn(len(region.AvailabilityZones) > 2))
		elbs[i] = fmt.Sprintf(`"ELBAccountId": %q,`, region.ELBAccountId)
		fargates[i] = fmt.Sprintf(`"Fargate": %q`, yn(region.Fargate))
	}

	rnMax := max(rns, 0)
	efsMax := max(efss, 0)
	tazMax := max(tazs, 0)
	elbMax := max(elbs, 0)
	fargateMax := max(fargates, 0)

	for i := range names {
		f := fmt.Sprintf(`      %%-%ds { %%-%ds %%-%ds %%-%ds %%-%ds },`, rnMax, efsMax, tazMax, elbMax, fargateMax)
		fmt.Printf(f, rns[i], efss[i], tazs[i], elbs[i], fargates[i])
		fmt.Println()
	}

	return nil
}

func fetchRegions() (Regions, error) {
	rs := Regions{}

	data, err := exec.Command("aws", "ssm", "get-parameters-by-path", "--path", "/aws/service/global-infrastructure/regions", "--query", "Parameters[].Value"	).CombinedOutput()

	if err != nil {
		fmt.Printf("string(data): %+v\n", string(data))
		return nil, errors.WithStack(err)
	}

	var regions []string

	if err := json.Unmarshal(data, &regions); err != nil {
		return nil, errors.WithStack(err)
	}

	for _, region := range regions {
		rs[region] = Region{}
	}

	return rs, nil
}

func fetchAvailabilityZones(regions Regions) error {
	var azs []string

	for name, region := range regions {

		data, err := exec.Command("bash", "-c", fmt.Sprintf("aws ssm get-parameters-by-path --path /aws/service/global-infrastructure/regions/%s/availability-zones --query 'Parameters[].Value'", name)).CombinedOutput()

		if err != nil {
			return errors.WithStack(fmt.Errorf(string(data)))
		}

		if err := json.Unmarshal(data, &azs); err != nil {
			return errors.WithStack(err)
		}

		for _, az := range azs {
			region.AvailabilityZones = append(region.AvailabilityZones, az)
		}

		regions[name] = region
	}

	return nil
}

func fetchEFS(regions Regions) error {
	data, err := exec.Command("aws", "ssm", "get-parameters-by-path", "--path", "/aws/service/global-infrastructure/services/efs/regions", "--query", "Parameters[].Value"	).CombinedOutput()

	if err != nil {
		fmt.Printf("string(data): %+v\n", string(data))
		return errors.WithStack(err)
	}

	var efsRegions []string

	if err := json.Unmarshal(data, &efsRegions); err != nil {
		return errors.WithStack(err)
	}

	for _, efsRegion := range efsRegions {
		region := regions[efsRegion]
		region.EFS = true
		regions[efsRegion] = region
	}

	return nil
}

func fetchELBAccountIds(regions Regions) error {
	b := surf.NewBrowser()

	if err := b.Open("https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html"); err != nil {
		return errors.WithStack(err)
	}

	rows := b.Find("h2#access-logging-bucket-permissions~div.procedure table tr")

	if rows.Length() < 1 {
		return errors.WithStack(fmt.Errorf("no elb account ids found"))
	}

	rows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		name := strings.TrimSuffix(strings.TrimSpace(s.Find("td:nth-child(1)").Text()), "*")
		id := strings.TrimSpace(s.Find("td:nth-child(3)").Text())

		region := regions[name]
		region.ELBAccountId = id
		regions[name] = region
	})

	return nil
}

func fetchFargate(regions Regions) error {

	data, err := exec.Command("aws", "ssm", "get-parameters-by-path", "--path", "/aws/service/global-infrastructure/services/fargate/regions", "--query", "Parameters[].Value"	).CombinedOutput()

	if err != nil {
		fmt.Printf("string(data): %+v\n", string(data))
		return errors.WithStack(err)
	}

	var fargateRegions []string

	if err := json.Unmarshal(data, &fargateRegions); err != nil {
		return errors.WithStack(err)
	}

	for _, fargateRegion := range fargateRegions {
		region := regions[fargateRegion]
		region.Fargate = true
		regions[fargateRegion] = region
	}

	return nil
}

func printRegions(regions Regions) {
	rns := []string{}
	elbs := []string{}

	for name := range regions {
		rns = append(rns, name)
		elbs = append(elbs, regions[name].ELBAccountId)
	}

	sort.Strings(rns)

	rnmax := max(rns, 6)
	elmax := max(elbs, 12)

	fmt.Printf(fmt.Sprintf("\n%%-%ds  %%s  %%-5s  %%-%ds  %%s\n", rnmax, elmax), "region", "azs", "efs", "elbaccountid", "fargate")

	for _, name := range rns {
		r := regions[name]
		fmt.Printf(fmt.Sprintf("%%-%ds  %%3d  %%-5t  %%-%ds  %%t\n", rnmax, elmax), name, len(r.AvailabilityZones), r.EFS, r.ELBAccountId, r.Fargate)
	}
}

func max(ss []string, min int) int {
	m := min

	for _, s := range ss {
		if len(s) > m {
			m = len(s)
		}
	}

	return m
}

func yn(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}
