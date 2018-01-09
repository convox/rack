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
)

type Region struct {
	Ami               string
	AvailabilityZones []string
	EFS               bool
	ELBAccountId      string
}

type Regions map[string]Region

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	regions := Regions{}

	if err := fetchAmis(regions); err != nil {
		return err
	}

	if err := fetchAvailabilityZones(regions); err != nil {
		return err
	}

	if err := fetchEFS(regions); err != nil {
		return err
	}

	if err := fetchELBAccountIds(regions); err != nil {
		return err
	}

	names := []string{}

	for name := range regions {
		names = append(names, name)
	}

	sort.Strings(names)

	rns := make([]string, len(names))
	amis := make([]string, len(names))
	efss := make([]string, len(names))
	tazs := make([]string, len(names))
	elbs := make([]string, len(names))

	for i, name := range names {
		region := regions[name]

		taz := "No"
		if len(region.AvailabilityZones) > 2 {
			taz = "Yes"
		}

		// not all accounts have 3 azs in us-west-1
		if name == "us-west-1" {
			taz = "No"
		}

		efs := "No"
		if region.EFS {
			efs = "Yes"
		}

		rns[i] = fmt.Sprintf("%q:", name)
		amis[i] = fmt.Sprintf(`"Ami": %q,`, region.Ami)
		efss[i] = fmt.Sprintf(`"EFS": %q,`, efs)
		tazs[i] = fmt.Sprintf(`"ThirdAvailabilityZone": %q,`, taz)
		elbs[i] = fmt.Sprintf(`"ELBAccountId": %q`, region.ELBAccountId)
	}

	rnMax := max(rns)
	amiMax := max(amis)
	efsMax := max(efss)
	tazMax := max(tazs)
	elbMax := max(elbs)

	for i := range names {
		f := fmt.Sprintf(`      %%-%ds { %%-%ds %%-%ds %%-%ds %%-%ds },`, rnMax, amiMax, efsMax, tazMax, elbMax)
		fmt.Printf(f, rns[i], amis[i], efss[i], tazs[i], elbs[i])
		fmt.Println()
	}

	return nil
}

func fetchAmis(regions Regions) error {
	b := surf.NewBrowser()

	if err := b.Open("https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-optimized_AMI.html"); err != nil {
		return err
	}

	rows := b.Find("table#w524aac17c15c15c11 tr")

	if rows.Length() < 1 {
		return fmt.Errorf("no amis found")
	}

	rows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		name := s.Find("td:nth-child(1)").Text()
		ami := s.Find("td:nth-child(3)").Text()

		regions[name] = Region{Ami: ami}
	})

	return nil
}

func fetchAvailabilityZones(regions Regions) error {
	var azs struct {
		AvailabilityZones []struct {
			ZoneName string
		}
	}

	for name, region := range regions {
		data, err := exec.Command("bash", "-c", fmt.Sprintf("aws ec2 describe-availability-zones --region %s", name)).CombinedOutput()
		if err != nil {
			return err
		}

		if err := json.Unmarshal(data, &azs); err != nil {
			return err
		}

		for _, az := range azs.AvailabilityZones {
			region.AvailabilityZones = append(region.AvailabilityZones, az.ZoneName)
		}

		regions[name] = region
	}

	return nil
}

func fetchEFS(regions Regions) error {
	b := surf.NewBrowser()

	if err := b.Open("https://docs.aws.amazon.com/general/latest/gr/rande.html"); err != nil {
		return err
	}

	rows := b.Find("table#w114aab7d111b3 tr")

	if rows.Length() < 1 {
		return fmt.Errorf("no efs entries found")
	}

	rows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		name := strings.TrimSpace(s.Find("td:nth-child(2)").Text())

		region := regions[name]
		region.EFS = true
		regions[name] = region
	})

	return nil
}

func fetchELBAccountIds(regions Regions) error {
	b := surf.NewBrowser()

	if err := b.Open("https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html"); err != nil {
		return err
	}

	rows := b.Find("table#w377aac17b9c15b9c12b3b5b3 tr")

	if rows.Length() < 1 {
		return fmt.Errorf("no elb account ids found")
	}

	rows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		name := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
		id := strings.TrimSpace(s.Find("td:nth-child(3)").Text())

		if !strings.HasSuffix(name, "*") {
			region := regions[name]
			region.ELBAccountId = id
			regions[name] = region
		}
	})

	// temp fix until aws fixes docs
	region := regions["eu-west-3"]
	region.ELBAccountId = "009996457667"
	regions["eu-west-3"] = region

	return nil
}

func max(ss []string) int {
	m := 0

	for _, s := range ss {
		if len(s) > m {
			m = len(s)
		}
	}

	return m
}
