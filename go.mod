module github.com/metal-pod/gardener-extension-provider-metal

go 1.12

require (
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/ajeddeloh/yaml v0.0.0-20141224210557-6b16a5714269 // indirect
	github.com/coreos/container-linux-config-transpiler v0.9.0
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/coreos/ignition v0.34.0 // indirect
	github.com/gardener/gardener v0.34.0
	github.com/gardener/gardener-extensions v1.2.1-0.20200115130121-fabf2e89eae4
	github.com/gardener/machine-controller-manager v0.25.0
	github.com/go-logr/logr v0.1.0
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/golang/mock v1.3.1
	github.com/google/go-cmp v0.3.1
	github.com/google/uuid v1.1.1
	github.com/metal-pod/cloud-go v0.0.0-20191211160716-e58fa1ae107b
	github.com/metal-pod/metal-go v0.2.1-0.20200120090002-c1f47f341bf5
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/vincent-petithory/dataurl v0.0.0-20191104211930-d1553a71de50 // indirect
	k8s.io/api v0.16.6
	k8s.io/apiextensions-apiserver v0.16.6
	k8s.io/apimachinery v0.16.6
	k8s.io/apiserver v0.16.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.16.6
	k8s.io/component-base v0.16.6
	k8s.io/kubelet v0.16.6
	sigs.k8s.io/controller-runtime v0.4.0
)

replace (
	github.com/ajeddeloh/yaml => github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/gardener/external-dns-management => github.com/gardener/external-dns-management v0.7.5
	github.com/gardener/machine-controller-manager => github.com/metal-pod/machine-controller-manager v0.0.0-20190801141331-4e2b75ebc6c0
	k8s.io/client-go => k8s.io/client-go v0.16.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.6
)
