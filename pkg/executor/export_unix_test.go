//go:build !windows

package executor

// SetupRawModeIfTTYForTest exposes setupRawModeIfTTY for tests.
func SetupRawModeIfTTYForTest(fd int) (func() error, error) {
	return setupRawModeIfTTY(fd)
}
