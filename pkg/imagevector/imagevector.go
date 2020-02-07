//go:generate packr2

package imagevector

import (
	"strings"

	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/gobuffalo/packr/v2"

	"k8s.io/apimachinery/pkg/util/runtime"
)

var imageVector imagevector.ImageVector

func init() {
	box := packr.New("charts", "../../charts")

	imagesYaml, err := box.FindString("images.yaml")
	runtime.Must(err)

	imageVector, err = imagevector.Read(strings.NewReader(imagesYaml))
	runtime.Must(err)

	imageVector, err = imagevector.WithEnvOverride(imageVector)
	runtime.Must(err)
}

// ImageVector is the image vector that contains all the needed images.
func ImageVector() imagevector.ImageVector {
	return imageVector
}
