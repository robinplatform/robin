package process

import "robinplatform.dev/internal/model"

type RHandle struct {
	db model.RHandle[Process]
}

func (manager *ProcessManager) ReadHandle() RHandle {
	return RHandle{db: manager.db.ReadHandle()}
}

func (r *RHandle) Close() {
	r.db.Close()
}

func (manager *ProcessManager) FindById(id ProcessId) (*Process, error) {
	r := manager.ReadHandle()
	defer r.Close()

	return r.FindById(id)
}

func (manager *ProcessManager) IsAlive(id ProcessId) bool {
	r := manager.ReadHandle()
	defer r.Close()

	return r.IsAlive(id)
}
