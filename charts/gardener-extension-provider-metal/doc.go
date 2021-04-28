//go:generate ../../vendor/github.com/gardener/gardener/hack/generate-controller-registration.sh provider-metal . ../../VERSION ../../example/controller-registration.yaml ControlPlane:metal Infrastructure:metal Worker:metal

// Package chart enables go:generate support for generating the correct controller registration.
package chart
