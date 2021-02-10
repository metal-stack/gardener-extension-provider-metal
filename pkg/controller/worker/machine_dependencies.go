package worker

import "context"

// DeployMachineDependencies implements genericactuator.WorkerDelegate.
func (w *workerDelegate) DeployMachineDependencies(_ context.Context) error {
	return nil
}

// CleanupMachineDependencies implements genericactuator.WorkerDelegate.
func (w *workerDelegate) CleanupMachineDependencies(_ context.Context) error {
	return nil
}
