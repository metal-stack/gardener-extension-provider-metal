package worker

import (
	"context"
)

// DeployMachineDependencies implements genericactuator.WorkerDelegate.
func (w *workerDelegate) DeployMachineDependencies(_ context.Context) error {
	return nil
}

// CleanupMachineDependencies implements genericactuator.WorkerDelegate.
func (w *workerDelegate) CleanupMachineDependencies(ctx context.Context) error {
	return nil
}

// PreReconcileHook implements genericactuator.WorkerDelegate.
func (w *workerDelegate) PreReconcileHook(ctx context.Context) error {
	return nil
}

// PostReconcileHook implements genericactuator.WorkerDelegate.
func (w *workerDelegate) PostReconcileHook(ctx context.Context) error {
	return nil
}

// PreDeleteHook implements genericactuator.WorkerDelegate.
func (w *workerDelegate) PreDeleteHook(_ context.Context) error {
	return nil
}

// PostDeleteHook implements genericactuator.WorkerDelegate.
func (w *workerDelegate) PostDeleteHook(ctx context.Context) error {
	return nil
}
