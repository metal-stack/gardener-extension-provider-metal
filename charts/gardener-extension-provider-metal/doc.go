//go:generate sh -c "bash $GARDENER_HACK_DIR/generate-controller-registration.sh provider-metal . $(cat ../../VERSION) ../../example/controller-registration.yaml ControlPlane:metal Infrastructure:metal Worker:metal"

// Package chart enables go:generate support for generating the correct controller registration.
package chart
