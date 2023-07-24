package loader

import (
	"os"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/install"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
)

var (
	Codec  runtime.Codec
	Scheme *runtime.Scheme
)

func init() {
	Scheme = runtime.NewScheme()
	install.Install(Scheme)
	yamlSerializer := json.NewYAMLSerializer(json.DefaultMetaFactory, Scheme, Scheme)
	Codec = versioning.NewDefaultingCodecForScheme(
		Scheme,
		yamlSerializer,
		yamlSerializer,
		schema.GroupVersion{Version: "v1alpha1"},
		runtime.InternalGroupVersioner,
	)
}

// LoadFromFile takes a filename and de-serializes the contents into ControllerConfiguration object.
func LoadFromFile(filename string) (*config.ControllerConfiguration, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return Load(bytes)
}

// Load takes a byte slice and de-serializes the contents into ControllerConfiguration object.
// Encapsulates de-serialization without assuming the source is a file.
func Load(data []byte) (*config.ControllerConfiguration, error) {
	cfg := &config.ControllerConfiguration{}

	if len(data) == 0 {
		return cfg, nil
	}

	decoded, _, err := Codec.Decode(data, &schema.GroupVersionKind{Version: "v1alpha1", Kind: "Config"}, cfg)
	if err != nil {
		return nil, err
	}

	return decoded.(*config.ControllerConfiguration), nil
}
