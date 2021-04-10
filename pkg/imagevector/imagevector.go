package imagevector

import (
	"strings"

	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/charts"

	"k8s.io/apimachinery/pkg/util/runtime"
)

var imageVector imagevector.ImageVector

func init() {
	var err error
	imageVector, err = imagevector.Read(strings.NewReader(charts.ImagesYAML))
	runtime.Must(err)

	imageVector, err = imagevector.WithEnvOverride(imageVector)
	runtime.Must(err)
}

// ImageVector is the image vector that contains all the needed images.
func ImageVector() imagevector.ImageVector {
	return imageVector
}
