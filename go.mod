module github.com/metal-stack/gardener-extension-provider-metal

go 1.27

require (
	github.com/Masterminds/semver/v3 v3.5.0
	github.com/coreos/go-systemd/v22 v22.7.0
	github.com/elastic/crd-ref-docs v0.3.0
	github.com/gardener/etcd-druid/api v0.36.4
	github.com/gardener/gardener v1.145.4
	github.com/gardener/gardener-extension-networking-calico v1.53.1
	github.com/gardener/gardener-extension-networking-cilium v1.45.2
	github.com/gardener/gardener/pkg/apis v1.145.4
	github.com/gardener/machine-controller-manager v0.61.3
	github.com/go-logr/logr v1.4.4
	github.com/go-openapi/strfmt v0.27.2
	github.com/google/go-cmp v0.7.0
	github.com/metal-stack/duros-controller v0.12.2
	github.com/metal-stack/firewall-controller-manager v0.6.1
	github.com/metal-stack/firewall-controller/v2 v2.5.0
	github.com/metal-stack/metal-go v0.45.0
	github.com/metal-stack/metal-lib v0.26.3
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/ginkgo/v2 v2.33.0
	github.com/onsi/gomega v1.44.0
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.12.1
	go.uber.org/mock v0.6.0
	golang.org/x/tools v0.50.0
	k8s.io/api v0.35.5
	k8s.io/apiextensions-apiserver v0.35.5
	k8s.io/apimachinery v0.35.5
	k8s.io/autoscaler/vertical-pod-autoscaler v1.6.0
	k8s.io/client-go v0.35.5
	k8s.io/code-generator v0.35.5
	k8s.io/component-base v0.35.5
	k8s.io/kubelet v0.35.5
	sigs.k8s.io/controller-runtime v0.23.3
	sigs.k8s.io/controller-tools v0.20.1
	sigs.k8s.io/yaml v1.6.0
)

require (
	cel.dev/expr v0.25.2 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20240806141605-e8a1dd7889d6 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/PaesslerAG/gval v1.2.4 // indirect
	github.com/PaesslerAG/jsonpath v0.1.2-0.20240726212847-3a740cf7976f // indirect
	github.com/VictoriaMetrics/VictoriaLogs v1.36.2-0.20251008164716-21c0fb3de84d // indirect
	github.com/VictoriaMetrics/VictoriaMetrics v1.131.0 // indirect
	github.com/VictoriaMetrics/easyproto v0.1.4 // indirect
	github.com/VictoriaMetrics/metrics v1.40.2 // indirect
	github.com/VictoriaMetrics/metricsql v0.84.8 // indirect
	github.com/VictoriaMetrics/operator/api v0.66.1 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.7 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.17 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/brunoga/deep v1.3.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-oidc/v3 v3.20.0 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/elliotchance/orderedmap/v3 v3.1.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fluent/fluent-operator/v3 v3.7.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/gardener/cert-management v0.23.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/analysis v0.25.5 // indirect
	github.com/go-openapi/errors v0.22.8 // indirect
	github.com/go-openapi/jsonpointer v1.0.0 // indirect
	github.com/go-openapi/jsonreference v1.0.0 // indirect
	github.com/go-openapi/loads v0.25.0 // indirect
	github.com/go-openapi/runtime v0.33.0 // indirect
	github.com/go-openapi/runtime/server-middleware v0.32.1 // indirect
	github.com/go-openapi/spec v0.22.9 // indirect
	github.com/go-openapi/swag v0.26.1 // indirect
	github.com/go-openapi/swag/cmdutils v0.26.1 // indirect
	github.com/go-openapi/swag/conv v0.28.0 // indirect
	github.com/go-openapi/swag/fileutils v0.28.0 // indirect
	github.com/go-openapi/swag/jsonname v0.26.1 // indirect
	github.com/go-openapi/swag/jsonutils v0.28.0 // indirect
	github.com/go-openapi/swag/loading v0.28.0 // indirect
	github.com/go-openapi/swag/mangling v0.28.0 // indirect
	github.com/go-openapi/swag/netutils v0.26.1 // indirect
	github.com/go-openapi/swag/pools v0.28.0 // indirect
	github.com/go-openapi/swag/stringutils v0.28.0 // indirect
	github.com/go-openapi/swag/typeutils v0.28.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.28.0 // indirect
	github.com/go-openapi/validate v0.26.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/cel-go v0.31.0 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.13-0.20220915233716-71ac16282d12 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0 // indirect
	github.com/labstack/echo/v4 v4.15.2 // indirect
	github.com/labstack/gommon v0.5.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc/v3 v3.0.6 // indirect
	github.com/lestrrat-go/jwx/v3 v3.2.0 // indirect
	github.com/lestrrat-go/option/v2 v2.0.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/metal-stack/security v0.9.7 // indirect
	github.com/miekg/dns v1.1.72 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.5.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/muhlemmer/gu v0.3.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nexucis/lamenv v0.5.2 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/open-telemetry/opentelemetry-operator/apis v0.153.0 // indirect
	github.com/perses/common v0.30.2 // indirect
	github.com/perses/perses v0.53.1 // indirect
	github.com/perses/perses-operator v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.91.0 // indirect
	github.com/prometheus/alertmanager v0.29.0 // indirect
	github.com/prometheus/client_golang v1.23.3-0.20260602051030-3537b20ac86b // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.68.1 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/prometheus/sigv4 v0.3.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fastjson v1.6.10 // indirect
	github.com/valyala/fastrand v1.1.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/valyala/gozstd v1.24.0 // indirect
	github.com/valyala/histogram v1.2.0 // indirect
	github.com/valyala/quicktemplate v1.8.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/zitadel/oidc/v3 v3.47.5 // indirect
	github.com/zitadel/schema v1.3.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.6 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/exp v0.0.0-20260811152304-ee035b5b010f // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/telemetry v0.0.0-20260908163034-4bcc4b2ee518 // indirect
	golang.org/x/term v0.46.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	helm.sh/helm/v4 v4.1.4 // indirect
	istio.io/api v1.30.0 // indirect
	istio.io/client-go v1.30.0 // indirect
	k8s.io/apiserver v0.35.5 // indirect
	k8s.io/gengo/v2 v2.0.0-20260408192533-25e2208e0dc3 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-aggregator v0.35.5 // indirect
	k8s.io/kube-openapi v0.0.0-20260520065146-aa012df4f4af // indirect
	k8s.io/metrics v0.35.5 // indirect
	k8s.io/pod-security-admission v0.35.5 // indirect
	k8s.io/utils v0.0.0-20260507154919-ff6756f316d2 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.35.0 // indirect
	sigs.k8s.io/gateway-api v1.5.0 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.0 // indirect
)
