package main

import (
	"os"

	"github.com/metal-stack/gardener-extension-provider-metal/cmd/gardener-extension-metal-hyper/app"

	"github.com/gardener/gardener/pkg/logger"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	runtimelog.SetLogger(logger.MustNewZapLogger(logger.InfoLevel, logger.FormatJSON))
	cmd := app.NewHyperCommand(signals.SetupSignalHandler())

	if err := cmd.Execute(); err != nil {
		runtimelog.Log.Error(err, "Error executing the main controller command")
		os.Exit(1)
	}
}
