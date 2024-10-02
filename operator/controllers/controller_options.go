package controllers

// Options are the config parameters for the controller
type Options struct {

	// number of reconcilers to run concurrently
	// multiple resources processed simultaneously, but each resource is handled by a single goroutine/thread in isolation
	MaxConcurrentReconciles int
}
