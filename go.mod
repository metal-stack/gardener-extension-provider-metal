module github.com/metal-pod/gardener-extension-provider-metal

go 1.12

require (
	cloud.google.com/go v0.37.1
	github.com/Azure/azure-pipeline-go v0.2.2 // indirect
	github.com/Azure/azure-storage-blob-go v0.7.0
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.20.0+incompatible // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20190723075400-e63e3f9dd712
	github.com/aliyun/aliyun-oss-go-sdk v2.0.1+incompatible
	github.com/appscode/jsonpatch v0.0.0-20190108182946-7c0e3b262f30
	github.com/aws/aws-sdk-go v1.12.79
	github.com/census-instrumentation/opencensus-proto v0.1.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/gardener/controller-manager-library v0.0.0-20190715130150-315e86b01963 // indirect
	github.com/gardener/external-dns-management v0.0.0-20190722114702-f6b12f6e4b43 // indirect
	github.com/gardener/gardener v0.0.0-20190812144748-dec8f3c7f288
	github.com/gardener/gardener-resource-manager v0.0.0-20190802153254-ea0dc5872b6a
	github.com/gardener/machine-controller-manager v0.0.0-20190606071036-119056ee3fdd
	github.com/go-ini/ini v1.44.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/gobuffalo/logger v1.0.1 // indirect
	github.com/gobuffalo/packd v0.3.0 // indirect
	github.com/gobuffalo/packr v1.25.0
	github.com/gobuffalo/packr/v2 v2.1.0
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/golang/mock v1.2.0
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gophercloud/gophercloud v0.2.0
	github.com/gophercloud/utils v0.0.0-20190527093828-25f1b77b8c03
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/jetstack/cert-manager v0.6.2
	github.com/karrick/godirwalk v1.10.12 // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/packethost/packngo v0.0.0-20181217122008-b3b45f1b4979
	github.com/pierrec/lz4 v2.0.5+incompatible
	github.com/pkg/errors v0.8.1
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/net v0.0.0-20190724013045-ca1201d0de80 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20190712062909-fae7ac547cb7 // indirect
	golang.org/x/tools v0.0.0-20190624180213-70d37148ca0c // indirect
	google.golang.org/api v0.2.0
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/genproto v0.0.0-20190716160619-c506a9f90610 // indirect
	google.golang.org/grpc v1.22.0 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.0.0-20190311093542-50b561225d70
	k8s.io/component-base v0.0.0-20190314000054-4a91899592f4
	k8s.io/helm v2.14.2+incompatible
	k8s.io/klog v0.3.3 // indirect
	k8s.io/kube-aggregator v0.0.0-20190314000639-da8327669ac5
	k8s.io/kube-openapi v0.0.0-20190722073852-5e22f3d471e6 // indirect
	k8s.io/kubelet v0.0.0-20190314002251-f6da02f58325
	k8s.io/utils v0.0.0-20190712204705-3dccf664f023 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab //kubernetes-1.14.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed // kubernetes-1.14.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190313205120-d7deff9243b1 // kubernetes-1.14.0
	k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible // kubernetes-1.14.0
)
