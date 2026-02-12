//go:build !windows

package main

// addDefenderExclusion is a no-op on non-Windows platforms.
func addDefenderExclusion() error {
	return nil
}
