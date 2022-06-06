module github.com/trufflesecurity/trufflehog/v3

go 1.18

replace github.com/jpillora/overseer => github.com/trufflesecurity/overseer v1.1.7-custom5

replace github.com/zricethezav/gitleaks/v8 => github.com/trufflesecurity/gitleaks/v8 v8.6.1-custom3

require (
	cloud.google.com/go/secretmanager v1.4.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.11
	github.com/aws/aws-sdk-go v1.44.20
	github.com/bill-rich/go-syslog v0.0.0-20220413021637-49edb52a574c
	github.com/bitfinexcom/bitfinex-api-go v0.0.0-20210608095005-9e0b26f200fb
	github.com/bradleyfalzon/ghinstallation/v2 v2.0.4
	github.com/crewjam/rfc5424 v0.1.0
	github.com/envoyproxy/protoc-gen-validate v0.6.7
	github.com/fatih/color v1.13.0
	github.com/felixge/fgprof v0.9.2
	github.com/gitleaks/go-gitdiff v0.7.6
	github.com/go-errors/errors v1.4.2
	github.com/go-git/go-git/v5 v5.4.2
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/go-github/v42 v42.0.0
	github.com/gorilla/mux v1.8.0
	github.com/h2non/filetype v1.1.3
	github.com/hashicorp/go-retryablehttp v0.7.1
	github.com/joho/godotenv v1.4.0
	github.com/jpillora/overseer v1.1.6
	github.com/kylelemons/godebug v1.1.0
	github.com/mattn/go-colorable v0.1.12
	github.com/paulbellamy/ratecounter v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/razorpay/razorpay-go v0.0.0-20210728161131-0341409a6ab2
	github.com/rs/zerolog v1.26.1
	github.com/sergi/go-diff v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.1
	github.com/tailscale/depaware v0.0.0-20210622194025-720c4b409502
	github.com/xanzy/go-gitlab v0.65.0
	github.com/zricethezav/gitleaks/v8 v8.5.2
	golang.org/x/crypto v0.0.0-20211215165025-cf75a172585e
	golang.org/x/net v0.0.0-20220325170049-de3da57026de
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/genproto v0.0.0-20220405205423-9d709892a2bf
	google.golang.org/protobuf v1.28.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/h2non/gock.v1 v1.1.2
)

require (
	cloud.google.com/go v0.100.2 // indirect
	cloud.google.com/go/compute v1.5.0 // indirect
	cloud.google.com/go/iam v0.3.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.24 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.18 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.5 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/golang-jwt/jwt/v4 v4.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/go-github/v41 v41.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/pprof v0.0.0-20211214055906-6f57359322fd // indirect
	github.com/googleapis/gax-go/v2 v2.2.0 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jpillora/s3 v1.1.4 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/pkg/diff v0.0.0-20200914180035-5b29258ca4f7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/mod v0.5.0 // indirect
	golang.org/x/sys v0.0.0-20220328115105-d36c6a25d886 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.1.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.74.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/grpc v1.45.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
