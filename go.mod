module github.com/metal-pod/gardener-extension-provider-metal

go 1.12

require (
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/gardener/gardener v0.0.0-20190827050434-cdafc6bd869f
	github.com/gardener/gardener-extensions v0.0.0-20190828072712-cf446972f37d
	github.com/gardener/machine-controller-manager v0.0.0-20190606071036-119056ee3fdd
	github.com/go-logr/logr v0.1.0
	github.com/gobuffalo/packr/v2 v2.5.2
	github.com/golang/mock v1.3.1
	github.com/google/uuid v1.1.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/metal-pod/metal-go v0.0.0-20190830132640-c0f862d8039a
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0 // indirect
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7 // indirect
	golang.org/x/sys v0.0.0-20190813064441-fde4db37ae7a // indirect
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/kubelet v0.0.0-20190314002251-f6da02f58325
	sigs.k8s.io/controller-runtime v0.2.0-beta.2
)

replace (
	git.apache.org/thrift => github.com/apache/thrift
	github.com/gardener/gardener => github.com/metal-pod/gardener v0.0.0-20190827131320-58dad7be7444
	github.com/gardener/machine-controller-manager => github.com/metal-pod/machine-controller-manager v0.0.0-20190801141331-4e2b75ebc6c0
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab //kubernetes-1.14.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed // kubernetes-1.14.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190313205120-d7deff9243b1 // kubernetes-1.14.0
	k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible // kubernetes-1.14.0
)
