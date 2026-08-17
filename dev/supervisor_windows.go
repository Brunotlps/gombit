//go:build windows

package dev

import (
	"os/exec"
)

// Windows has no POSIX process groups. Teardown uses `taskkill /T` so
// grandchildren (air rebuilds, npm/pnpm → node/vite) exit with the child.
// CREATE_NEW_PROCESS_GROUP is not used; it does not make Process.Kill
// terminate the tree.

func prepareProcessGroup(_ *exec.Cmd) {}

func signalProcessGroup(cmd *exec.Cmd) error {
	return taskkillTree(cmd, false)
}

func killProcessGroup(cmd *exec.Cmd) error {
	return taskkillTree(cmd, true)
}

func taskkillTree(cmd *exec.Cmd, force bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	args := windowsTaskkillArgs(cmd.Process.Pid, force)
	return exec.Command("taskkill", args...).Run() //nolint:gosec // fixed taskkill invocation
}
