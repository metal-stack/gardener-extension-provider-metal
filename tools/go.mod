module github.com/gardener/gardener/extensions/tools

go 1.13

require (
	github.com/go-critic/go-critic v0.3.5-0.20190210220443-ee9bf5809ead // indirect
	github.com/gobuffalo/packr/v2 v2.1.0 // indirect
	github.com/golang/mock v1.2.0 // indirect
	github.com/golangci/golangci-lint v1.16.1-0.20190425135923-692dacb773b7 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	sourcegraph.com/sourcegraph/go-diff v0.5.1-0.20190210232911-dee78e514455 // indirect
)

// the following replacements are required as long as these repos do not have valid versioning
replace (
	github.com/go-critic/go-critic => github.com/go-critic/go-critic v0.3.5-0.20190422201921-c3db6069acc5
	github.com/golangci/errcheck => github.com/golangci/errcheck v0.0.0-20181223084120-ef45e06d44b6
	github.com/golangci/go-tools => github.com/golangci/go-tools v0.0.0-20190318060251-af6baa5dc196
	github.com/golangci/gofmt => github.com/golangci/gofmt v0.0.0-20181222123516-0b8337e80d98
	github.com/golangci/gosec => github.com/golangci/gosec v0.0.0-20190211064107-66fb7fc33547
	github.com/golangci/lint-1 => github.com/golangci/lint-1 v0.0.0-20190420132249-ee948d087217
	mvdan.cc/unparam => mvdan.cc/unparam v0.0.0-20190209190245-fbb59629db34
)
