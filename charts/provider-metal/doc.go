//go:generate ../../hack/generate-controller-registration.sh provider-metal . ../../example/controller-registration.yaml Infrastructure:metal ControlPlane:metal Worker:metal

// Package chart enables go:generate support for generating the correct controller registration.
package chart
