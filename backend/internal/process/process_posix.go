//go:build !windows

package process

import (
	"fmt"
	"syscall"
)

func getProcessSysAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// Kill will kill the process with the given id (not PID), and remove it from
// the internal database.
func (w *WHandle) Kill(id ProcessId) error {
	procEntry, found := w.db.Find(findById(id))
	if !found {
		return processNotFound(id)
	}

	// We will not treat ESRCH as an error, since it means the process is already dead.
	if err := syscall.Kill(procEntry.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	if err := w.db.Delete(findById(id)); err != nil {
		return err
	}

	return nil
}
