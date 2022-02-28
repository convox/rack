module github.com/convox/rack

go 1.16

require (
	github.com/PuerkitoBio/goquery v1.1.0
	github.com/RackSec/srslog v0.0.0-20170920152354-4d2c753a4ee1
	github.com/aws/aws-lambda-go v1.2.0
	github.com/aws/aws-sdk-go v1.25.29
	github.com/boltdb/bolt v1.3.1
	github.com/convox/changes v0.0.0-20190306122126-bce25ca20c47
	github.com/convox/exec v0.0.0-20180905012044-cc13d277f897
	github.com/convox/logger v0.0.0-20180522214415-e39179955b52
	github.com/convox/stdapi v0.0.0-20190628182814-148bcf53d167
	github.com/convox/stdcli v0.0.0-20190326115454-b78bee159e98
	github.com/convox/stdsdk v0.0.0-20190422120437-3e80a397e377
	github.com/convox/version v0.0.0-20160822184233-ffefa0d565d2
	github.com/docker/docker v1.13.1
	github.com/docker/go-units v0.3.2
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.7.0
	github.com/fsouza/go-dockerclient v0.0.0-20160427172547-1d4f4ae73768
	github.com/gobuffalo/packr v1.22.0
	github.com/gobwas/glob v0.2.3
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/websocket v1.4.1
	github.com/headzoo/surf v1.0.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/miekg/dns v1.1.25
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mweagle/Sparta v0.8.1-0.20171126182155-ead2872585dc
	github.com/pkg/errors v0.8.1
	github.com/segmentio/analytics-go v2.0.1-0.20160426181448-2d840d861c32+incompatible
	github.com/stretchr/testify v1.3.0
	github.com/stvp/rollbar v0.5.1
	github.com/twmb/algoimpl v0.0.0-20170717182524-076353e90b94
	golang.org/x/crypto v0.0.0-20190923035154-9ee001bba392
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/metrics v0.0.0-20180628054111-6f051017e10b
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.5 // indirect
	github.com/Sirupsen/logrus v1.0.3 // indirect
	github.com/andybalholm/cascadia v0.0.0-20161224141413-349dd0209470 // indirect
	github.com/bearsh/hid v1.4.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/convox/go-u2fhost v0.0.0-20220210143516-c133f566e496
	github.com/convox/inotify v0.0.0-20170313035821-b56f5149b5c6 // indirect
	github.com/docker/spdystream v0.0.0-20170912183627-bc6354cbbc29 // indirect
	github.com/elazarl/goproxy v0.0.0-20210801061803-8e322dfb79c4 // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/hashicorp/golang-lru v0.0.0-20180201235237-0fb14efe8c47 // indirect
	github.com/headzoo/ut v0.0.0-20181013193318-a13b5a7a02ca // indirect
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/moby/moby v1.13.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/segmentio/backo-go v0.0.0-20160424052352-204274ad699c // indirect
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/kube-openapi v0.0.0-20180731170545-e3762e86a74c // indirect
)

replace gopkg.in/yaml.v2 => github.com/ddollar/yaml v0.0.0-20180504010936-3fb95e32dd8a

replace github.com/fsouza/go-dockerclient => github.com/heronrs/go-dockerclient v0.0.1
