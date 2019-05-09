package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
)

type Region struct {
	Ami string
	EFS bool
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

	// fmt.Printf("regions = %+v\n", regions)
	// return nil

	if err := fetchEFS(regions); err != nil {
		return err
	}

	names := []string{}

	for name := range regions {
		names = append(names, name)
	}

	sort.Strings(names)

	fmt.Println("  Regions:")

	for _, name := range names {
		region := regions[name]

		if region.Ami == "" {
			continue
		}

		fmt.Printf("    %s:\n", name)
		fmt.Printf("      AMI: %s\n", region.Ami)
		fmt.Printf("      EFS: %s\n", yn(region.EFS))
	}

	// rns := make([]string, len(names))
	// amis := make([]string, len(names))
	// efss := make([]string, len(names))
	// tazs := make([]string, len(names))
	// elbs := make([]string, len(names))
	// fargates := make([]string, len(names))

	// for i, name := range names {
	//   region := regions[name]

	//   rns[i] = fmt.Sprintf("%q:", name)
	//   amis[i] = fmt.Sprintf(`"Ami": %q,`, region.Ami)
	//   efss[i] = fmt.Sprintf(`"EFS": %q,`, yn(region.EFS))
	// }

	// rnMax := max(rns)
	// amiMax := max(amis)
	// efsMax := max(efss)

	// for i := range names {
	//   if regions[names[i]].Ami == "" {
	//     continue
	//   }

	//   f := fmt.Sprintf(`      %%-%ds { %%-%ds %%-%ds %%-%ds %%-%ds %%-%ds },`, rnMax, amiMax, efsMax, tazMax, elbMax, fargateMax)
	//   fmt.Printf(f, rns[i], amis[i], efss[i], tazs[i], elbs[i], fargates[i])
	//   fmt.Println()
	// }

	return nil
}

func fetchAmis(regions Regions) error {
	b := surf.NewBrowser()

	if err := b.Open("https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html"); err != nil {
		return err
	}

	// rows := b.Find("#main-content dd[data-tab*=\"kubernetes-version-1.12\"] .table-contents table:nth-child(1) tr")
	rows := b.Find("#main-content dd[data-tab*=\"kubernetes-version-1.12\"] .table-contents tr")

	if rows.Length() < 1 {
		return fmt.Errorf("no amis found")
	}

	rows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		name := s.Find("td:nth-child(1)")
		ami := s.Find("td:nth-child(2)").Text()

		if name.Text() == "" {
			return
		}

		region := name.Find("code").Text()

		regions[region] = Region{Ami: ami}
	})

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
