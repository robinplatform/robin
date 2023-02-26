//go:build windows

package process

import (
	"fmt"
	"os"
	"syscall"

	"robinplatform.dev/internal/log"
)

func getProcessSysAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

// Kill will kill the process with the given id (not PID), and remove it from
// the internal database.
// TODO: Make this work on windows
func (m *ProcessManager[Meta]) Kill(id ProcessId) error {
	w := m.db.WriteHandle()
	defer w.Close()

	procEntry, found := w.Find(findById[Meta](id))
	if !found {
		return processNotFound(id)
	}

	// On Windows, the failure to find a process represents that the process
	// is not running, so in this case, we can skip sending a kill signal and just
	// remove the process from the database.
	osProcess, err := os.FindProcess(procEntry.Pid)
	if err != nil {
		logger.Warn("Failed to find process, ignoring kill request", log.Ctx{
			"pid": procEntry.Pid,
		})
	} else {
		if err := osProcess.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	if err := w.Delete(findById[Meta](id)); err != nil {
		return err
	}

	return nil
}