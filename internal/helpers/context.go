package helpers // This package assumes your contextkeys/keys.go is here

import (
	"context"
)

// GetVMID retrieves the VM ID from the context.
// It returns the VM ID and a boolean indicating if it was found.
func GetVMID(ctx context.Context) (string, bool) {
	vmID, ok := ctx.Value(VMIDKey).(string)
	return vmID, ok
}

// MustGetVMID retrieves the VM ID from the context or panics if not found.
// Use this if you are absolutely sure the middleware has run.
func MustGetVMID(ctx context.Context) string {
	vmID, ok := GetVMID(ctx)
	if !ok {
		panic("VM ID not found in context. Ensure DomainMiddleware is used.")
	}
	return vmID
}

// GetVMDir retrieves the VM directory path from the context.
// It returns the VM directory path and a boolean indicating if it was found.
func GetVMDir(ctx context.Context) (string, bool) {
	vmDir, ok := ctx.Value(VMDirKey).(string)
	return vmDir, ok
}

// MustGetVMDir retrieves the VM directory path from the context or panics if not found.
// Use this if you are absolutely sure the middleware has run.
func MustGetVMDir(ctx context.Context) string {
	vmDir, ok := GetVMDir(ctx)
	if !ok {
		panic("VM directory not found in context. Ensure DomainMiddleware is used.")
	}
	return vmDir
}
