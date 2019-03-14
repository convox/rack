package k8s

const (
	BuildMax = 30
)

func (p *Provider) Workers() error {
	// go helpers.Tick(1*time.Hour, workerHandler(p.workerBuildCleanup))

	return nil
}

// func (p *Provider) workerBuildCleanup() error {
//   as, err := p.AppList()
//   if err != nil {
//     return err
//   }

//   for _, a := range as {
//     fmt.Printf("a = %+v\n", a)

//     build := ""

//     if a.Release != "" {
//       r, err := p.ReleaseGet(a.Name, a.Release)
//       if err != nil {
//         return err
//       }
//       build = r.Build
//     }

//     fmt.Printf("build = %+v\n", build)

//     bs, err := p.BuildList(a.Name, structs.BuildListOptions{Limit: options.Int(1000)})
//     if err != nil {
//       return err
//     }

//     if len(bs) > BuildMax {
//       for _, b := range bs[BuildMax:] {
//         fmt.Printf("b = %+v\n", b)
//       }
//     }

//     // rs, err := p.BuildList(a.Name, structs.BuildListOptions{})
//     // if err != nil {
//     //   return err
//     // }

//     // fmt.Printf("len(rs) = %+v\n", len(rs))
//   }

//   return nil
// }

// func workerHandler(fn func() error) func() {
//   return func() {
//     if err := fn(); err != nil {
//       fmt.Printf("err = %+v\n", err)
//     }
//   }
// }
