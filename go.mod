module github.com/metal-stack/gardener-extension-provider-metal

go 1.15

require (
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/ajeddeloh/yaml v0.0.0-00010101000000-000000000000 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/container-linux-config-transpiler v0.9.0
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/go-systemd/v22 v22.1.0
	github.com/coreos/ignition v0.35.0 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/gardener/etcd-druid v0.3.0
	github.com/gardener/gardener v1.11.1
	github.com/gardener/machine-controller-manager v0.35.2
	github.com/go-logr/logr v0.3.0
	github.com/gobuffalo/packr/v2 v2.8.0
	github.com/golang/mock v1.4.4
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.2
	github.com/imdario/mergo v0.3.11
	github.com/metal-stack/duros-controller v0.1.1
	github.com/metal-stack/firewall-controller v1.0.1
	github.com/metal-stack/machine-controller-manager-provider-metal v0.1.3
	github.com/metal-stack/metal-go v0.11.5
	github.com/metal-stack/metal-lib v0.6.9
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/vincent-petithory/dataurl v0.0.0-20191104211930-d1553a71de50 // indirect
	go4.org v0.0.0-20180809161055-417644f6feb5 // indirect
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/apiserver v0.18.8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.18.8
	k8s.io/component-base v0.18.8
	k8s.io/kubelet v0.18.8
	sigs.k8s.io/controller-runtime v0.6.4
)

replace (
	github.com/ajeddeloh/yaml => github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/gardener/gardener-resource-manager v0.13.1 => github.com/gardener/gardener-resource-manager v0.17.1
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/apiserver => k8s.io/apiserver v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
	k8s.io/component-base => k8s.io/component-base v0.18.8
	k8s.io/helm => k8s.io/helm v2.13.1+incompatible
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.8
)
