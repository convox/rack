package models

// type CronJob struct {
//   Name     string `yaml:"name"`
//   Schedule string `yaml:"schedule"`
//   Command  string `yaml:"command"`
//   Service  *manifest1.Service
//   App      *App
// }

// //CronJobs is a wrapper for sorting
// type CronJobs []CronJob

// func (a CronJobs) Len() int           { return len(a) }
// func (a CronJobs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
// func (a CronJobs) Less(i, j int) bool { return a[i].Name < a[j].Name }

// func NewCronJobFromLabel(key, value string) CronJob {
//   keySlice := strings.Split(key, ".")
//   name := keySlice[len(keySlice)-1]
//   tokens := strings.Fields(value)
//   cronjob := CronJob{
//     Name:     name,
//     Schedule: fmt.Sprintf("cron(%s *)", strings.Join(tokens[0:5], " ")),
//     Command:  strings.Join(tokens[5:], " "),
//   }
//   return cronjob
// }

// func (cr *CronJob) AppName() string {
//   return cr.App.Name
// }

// func (cr *CronJob) Process() string {
//   return cr.Service.Name
// }

// func (cr *CronJob) ShortName() string {
//   shortName := fmt.Sprintf("%s%s", strings.Title(cr.Service.Name), strings.Title(cr.Name))

//   reg, err := regexp.Compile("[^A-Za-z0-9]+")
//   if err != nil {
//     panic(err)
//   }

//   return reg.ReplaceAllString(shortName, "")
// }

// func (cr *CronJob) LongName() string {
//   prefix := fmt.Sprintf("%s-%s-%s", cr.App.StackName(), cr.Process(), cr.Name)
//   hash := sha256.Sum256([]byte(prefix))
//   suffix := "-" + base32.StdEncoding.EncodeToString(hash[:])[:7]

//   // $prefix-$suffix-schedule" needs to be <= 64 characters
//   if len(prefix) > 55-len(suffix) {
//     prefix = prefix[:55-len(suffix)]
//   }
//   return prefix + suffix
// }
