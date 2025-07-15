module github.com/metal-stack/gardener-extension-provider-metal

go 1.24.0

require (
	github.com/Masterminds/semver/v3 v3.3.1
	github.com/ahmetb/gen-crd-api-reference-docs v0.3.0
	github.com/coreos/go-systemd/v22 v22.5.0
	github.com/gardener/etcd-druid v0.27.0
	github.com/gardener/etcd-druid/api v0.28.0
	github.com/gardener/gardener v1.116.4
	github.com/gardener/gardener-extension-networking-calico v1.42.0
	github.com/gardener/gardener-extension-networking-cilium v1.37.0
	github.com/gardener/machine-controller-manager v0.57.1
	github.com/go-logr/logr v1.4.3
	github.com/go-openapi/strfmt v0.23.0
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.7.0
	github.com/metal-stack/duros-controller v0.11.5
	github.com/metal-stack/firewall-controller-manager v0.5.1
	github.com/metal-stack/firewall-controller/v2 v2.3.8
	github.com/metal-stack/metal-go v0.41.2
	github.com/metal-stack/metal-lib v0.23.1
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.37.0
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/stretchr/testify v1.10.0
	k8s.io/api v0.33.2
	k8s.io/apiextensions-apiserver v0.33.2
	k8s.io/apimachinery v0.33.2
	k8s.io/autoscaler/vertical-pod-autoscaler v1.4.1
	k8s.io/client-go v0.33.2
	k8s.io/code-generator v0.33.2
	k8s.io/component-base v0.33.2
	k8s.io/kubelet v0.33.2
	sigs.k8s.io/controller-runtime v0.21.0
	sigs.k8s.io/yaml v1.5.0
)

replace (
	k8s.io/code-generator => k8s.io/code-generator v0.29.5
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.20.2
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-oidc/v3 v3.14.1 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fluent/fluent-operator/v3 v3.3.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.8.0 // indirect
	github.com/gardener/cert-management v0.17.5 // indirect
	github.com/go-jose/go-jose/v4 v4.1.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/runtime v0.28.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/pprof v0.0.0-20250607225305-033d6d78b36a // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/ironcore-dev/vgopath v0.1.8 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.6 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/metal-stack/security v0.9.4 // indirect
	github.com/miekg/dns v1.1.66 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/onsi/ginkgo/v2 v2.23.4 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.83.0 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.mongodb.org/mongo-driver v1.17.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/mock v0.5.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.3 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	helm.sh/helm/v3 v3.18.3 // indirect
	istio.io/api v1.26.2 // indirect
	istio.io/client-go v1.26.2 // indirect
	k8s.io/gengo v0.0.0-20250604051438-85fd79dbfd9f // indirect
	k8s.io/gengo/v2 v2.0.0-20250604051438-85fd79dbfd9f // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-aggregator v0.33.2 // indirect
	k8s.io/kube-openapi v0.0.0-20250610211856-8b98d1ed966a // indirect
	k8s.io/metrics v0.33.2 // indirect
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397 // indirect
	sigs.k8s.io/controller-runtime/tools/setup-envtest v0.0.0-20231015215740-bf15e44028f9 // indirect
	sigs.k8s.io/controller-tools v0.18.0 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.7.0 // indirect
)
