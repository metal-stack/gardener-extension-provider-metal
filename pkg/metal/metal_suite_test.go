package metal_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMetal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metal Suite")
}
