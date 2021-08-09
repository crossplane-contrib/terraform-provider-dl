module github.com/ulucinar/terraform-provider-dl

go 1.14

require (
	github.com/Masterminds/semver v1.5.0
	github.com/alecthomas/kong v0.2.15
	github.com/crossplane-contrib/terraform-provider-gen v0.0.0-20210317192616-e9255f2c6e7d
	github.com/hashicorp/terraform v0.13.5
	github.com/hashicorp/terraform-svchost v0.0.0-20191011084731-65d371908596
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.4.1
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
