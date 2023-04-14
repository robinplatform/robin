package health

import (
	"os"
	"runtime"
	"syscall"

	"robinplatform.dev/internal/log"
)

type ProcessHealthCheck struct {
}

func (ProcessHealthCheck) Check(info RunningProcessInfo) bool {
	return false
}

func PidIsAlive(pid int) bool {
	// TODO: check the actual error, it might've been a permission error
	// or something else.
	osProcess, err := os.FindProcess(pid)
	if err != nil {
		logger.Debug("got error when checking process alive", log.Ctx{
			"procIsNil": osProcess == nil,
			"err":       err.Error(),
		})
		return false
	}

	// It turns out, `Release` is super duper important on Windows. Without calling release,
	// the underlying Windows handle doesn't get closed, and the process stays in the "running"
	// state, at least for the purpose of this check. This isn't a problem on unix, as Release is essentially
	// a no-op there.
	defer osProcess.Release()

	// On windows, if we located a process, it's alive.
	// On other platforms, we only have a handle, and need to send a signal
	// to see if it's alive.
	if runtime.GOOS == "windows" {
		return true
	}

	return osProcess.Signal(syscall.Signal(0)) == nil
}
