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
	Fargate           bool
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

	if err := fetchFargate(regions); err != nil {
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
	fargates := make([]string, len(names))

	for i, name := range names {
		region := regions[name]

		rns[i] = fmt.Sprintf("%q:", name)
		amis[i] = fmt.Sprintf(`"Ami": %q,`, region.Ami)
		efss[i] = fmt.Sprintf(`"EFS": %q,`, yn(region.EFS))
		tazs[i] = fmt.Sprintf(`"ThirdAvailabilityZone": %q,`, yn(len(region.AvailabilityZones) > 2))
		elbs[i] = fmt.Sprintf(`"ELBAccountId": %q,`, region.ELBAccountId)
		fargates[i] = fmt.Sprintf(`"Fargate": %q`, yn(region.Fargate))
	}

	rnMax := max(rns)
	amiMax := max(amis)
	efsMax := max(efss)
	tazMax := max(tazs)
	elbMax := max(elbs)
	fargateMax := max(fargates)

	for i := range names {
		if regions[names[i]].Ami == "" {
			continue
		}

		f := fmt.Sprintf(`      %%-%ds { %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds },`, rnMax, amiMax, efsMax, tazMax, elbMax, fargateMax)
		fmt.Printf(f, rns[i], amis[i], efss[i], tazs[i], elbs[i], fargates[i])
		fmt.Println()
	}

	return nil
}

func fetchAmis(regions Regions) error {
	b := surf.NewBrowser()

	if err := b.Open("https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-optimized_AMI.html"); err != nil {
		return err
	}

	rows := b.Find("#main-content .table-contents table:nth-child(1) tr")

	if rows.Length() < 1 {
		return fmt.Errorf("no amis found")
	}

	rows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		name := s.Find("td:nth-child(1)").Text()
		ami := s.Find("td:nth-child(3)").Text()

		if strings.HasPrefix(name, "us-gov-") {
			return
		}

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

	rows := b.Find("h2#elasticfilesystem-region+.table .table-contents table tr")

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

	rows := b.Find("h2#access-logging-bucket-permissions~ol:nth-of-type(1) table tr")

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

	return nil
}

func fetchFargate(regions Regions) error {
	b := surf.NewBrowser()

	if err := b.Open("https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html"); err != nil {
		return err
	}

	rows := b.Find("h1#AWS_Fargate~.table:nth-of-type(3) tr")

	if rows.Length() < 1 {
		return fmt.Errorf("no fargate regions found")
	}

	rows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		name := strings.TrimSpace(s.Find("td:nth-child(2)").Text())

		if !strings.HasSuffix(name, "*") {
			region := regions[name]
			region.Fargate = true
			regions[name] = region
		}
	})

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

func yn(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}
