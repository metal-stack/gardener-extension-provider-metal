module github.com/metal-pod/gardener-extension-provider-metal

go 1.12

require (
	github.com/Bowery/prompt v0.0.0-20190419144237-972d0ceb96f5 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/dchest/safefile v0.0.0-20151022103144-855e8d98f185 // indirect
	github.com/gardener/gardener v0.0.0-20190812144748-dec8f3c7f288
	github.com/gardener/gardener-extensions v0.0.0-20190814053833-ca68fc5b800b
	github.com/gardener/machine-controller-manager v0.0.0-20190606071036-119056ee3fdd
	github.com/go-logr/logr v0.1.0
	github.com/gobuffalo/packr/v2 v2.1.0
	github.com/golang/mock v1.2.0
	github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf // indirect
	github.com/google/uuid v1.1.1
	github.com/kardianos/govendor v1.0.9 // indirect
	github.com/metal-pod/metal-go v0.0.0-20190813124841-17be76055943
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/kubelet v0.0.0-20190314002251-f6da02f58325
	sigs.k8s.io/controller-runtime v0.2.0-beta.2
)

replace (
	github.com/gardener/machine-controller-manager => github.com/metal-pod/machine-controller-manager v0.0.0-20190801141331-4e2b75ebc6c0
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab //kubernetes-1.14.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed // kubernetes-1.14.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190313205120-d7deff9243b1 // kubernetes-1.14.0
	k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible // kubernetes-1.14.0
)
