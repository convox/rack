module github.com/convox/rack

go 1.23.0

toolchain go1.23.3

require (
	github.com/RackSec/srslog v0.0.0-20170920152354-4d2c753a4ee1
	github.com/adhocore/gronx v1.6.5
	github.com/aws/aws-lambda-go v1.37.0
	github.com/aws/aws-sdk-go v1.25.29
	github.com/boltdb/bolt v1.3.1
	github.com/convox/changes v0.0.0-20250404235107-ec2c7374fea5
	github.com/convox/exec v0.0.0-20180905012044-cc13d277f897
	github.com/convox/logger v0.0.0-20180522214415-e39179955b52
	github.com/convox/stdapi v1.1.3-0.20221110171947-8d98f61e61ed
	github.com/convox/stdcli v0.0.0-20230203181735-23ed17b69b51
	github.com/convox/stdsdk v0.0.2
	github.com/convox/version v0.0.0-20160822184233-ffefa0d565d2
	github.com/docker/docker v27.5.0+incompatible
	github.com/docker/go-units v0.5.0
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.15.0
	github.com/fsouza/go-dockerclient v0.0.0-20160427172547-1d4f4ae73768
	github.com/gobuffalo/packr v1.22.0
	github.com/gobwas/glob v0.2.3
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.0
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/patternmatcher v0.6.0
	github.com/mweagle/Sparta v0.8.1-0.20171126182155-ead2872585dc
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/segmentio/analytics-go v2.0.1-0.20160426181448-2d840d861c32+incompatible
	github.com/stretchr/testify v1.9.0
	github.com/stvp/rollbar v0.5.1
	github.com/twmb/algoimpl v0.0.0-20170717182524-076353e90b94
	golang.org/x/crypto v0.38.0
	gopkg.in/cheggaaa/pb.v1 v1.0.28
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/adrg/xdg v0.2.1 // indirect
	github.com/bearsh/hid v1.4.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/convox/go-u2fhost v0.0.0-20220210143516-c133f566e496
	github.com/convox/inotify v0.0.0-20170313035821-b56f5149b5c6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/cli v27.5.0+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/engine v1.4.2-0.20190717161051-705d9623b7c1 // indirect
	github.com/gobuffalo/buffalo-plugins v1.11.0 // indirect
	github.com/gobuffalo/envy v1.6.12 // indirect
	github.com/gobuffalo/events v1.1.9 // indirect
	github.com/gobuffalo/flect v0.0.0-20190117212819-a62e61d96794 // indirect
	github.com/gobuffalo/genny v0.0.0-20190112155932-f31a84fcacf5 // indirect
	github.com/gobuffalo/logger v0.0.0-20181127160119-5b956e21995c // indirect
	github.com/gobuffalo/mapi v1.0.1 // indirect
	github.com/gobuffalo/meta v0.0.0-20190120163247-50bbb1fa260d // indirect
	github.com/gobuffalo/packd v0.0.0-20181212173646-eca3b8fd6687 // indirect
	github.com/gobuffalo/packr/v2 v2.0.0-rc.15 // indirect
	github.com/gobuffalo/syncx v0.0.0-20181120194010-558ac7de985f // indirect
	github.com/google/go-containerregistry v0.20.3
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gorilla/sessions v1.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af // indirect
	github.com/joho/godotenv v1.3.0 // indirect
	github.com/karrick/godirwalk v1.7.8 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/markbates/oncer v0.0.0-20181203154359-bf2de49a0be2 // indirect
	github.com/markbates/safe v1.0.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sebest/xff v0.0.0-20160910043805-6c115e0ffa35 // indirect
	github.com/segmentio/backo-go v0.0.0-20160424052352-204274ad699c // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/vbatts/tar-split v0.11.6 // indirect
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/tools v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.5.2 // indirect
)

replace gopkg.in/yaml.v2 => github.com/ddollar/yaml v0.0.0-20180504010936-3fb95e32dd8a

replace github.com/fsouza/go-dockerclient => github.com/heronrs/go-dockerclient v0.0.1
