//go:build !darwin

package dock

// Hide no-op on non-darwin
func Hide() {}

// Show no-op on non-darwin
func Show() {}
