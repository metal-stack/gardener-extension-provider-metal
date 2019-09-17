module github.com/gardener/gardener-extensions/tools

go 1.13

require (
	github.com/go-critic/go-critic ee9bf5809ead // indirect
	github.com/gobuffalo/packr/v2 v2.1.0 // indirect
	github.com/golang/mock v1.2.0 // indirect
	github.com/golangci/golangci-lint v1.16.1-0.20190425135923-692dacb773b7 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	sourcegraph.com/sourcegraph/go-diff v0.5.1-0.20190210232911-dee78e514455 // indirect
)

// the following replacements are required as long as these repos do not have valid versioning
replace (
	github.com/go-critic/go-critic => github.com/go-critic/go-critic c3db6069acc5
	github.com/golangci/errcheck => github.com/golangci/errcheck ef45e06d44b6
	github.com/golangci/go-tools => github.com/golangci/go-tools af6baa5dc196
	github.com/golangci/gofmt => github.com/golangci/gofmt 0b8337e80d98
	github.com/golangci/gosec => github.com/golangci/gosec 66fb7fc33547
	github.com/golangci/lint-1 => github.com/golangci/lint-1 ee948d087217
	mvdan.cc/unparam => mvdan.cc/unparam fbb59629db34
)