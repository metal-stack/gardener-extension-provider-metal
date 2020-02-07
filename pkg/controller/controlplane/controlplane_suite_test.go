package controlplane_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestControlplane(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metal Controlplane Suite")
}
