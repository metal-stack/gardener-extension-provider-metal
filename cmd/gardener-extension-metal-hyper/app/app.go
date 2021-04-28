package app

import (
	"context"

	admissionmetal "github.com/metal-stack/gardener-extension-provider-metal/cmd/gardener-extension-admission-metal/app"
	providermetal "github.com/metal-stack/gardener-extension-provider-metal/cmd/gardener-extension-provider-metal/app"

	"github.com/spf13/cobra"
)

// NewHyperCommand creates a new Hyper command consisting of all controllers under this repository.
func NewHyperCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "gardener-extension-hyper",
	}

	cmd.AddCommand(
		providermetal.NewControllerManagerCommand(ctx),
		admissionmetal.NewAdmissionCommand(ctx),
	)

	return cmd
}
