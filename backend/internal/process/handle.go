package process

import "robinplatform.dev/internal/model"

type RHandle struct {
	db model.RHandle[Process]
}

type WHandle struct {
	Read RHandle
	db   model.WHandle[Process]
}

func (manager *ProcessManager) ReadHandle() RHandle {
	return RHandle{db: manager.db.ReadHandle()}
}

func (manager *ProcessManager) WriteHandle() WHandle {
	db := manager.db.WriteHandle()
	return WHandle{
		Read: RHandle{db.UncloseableReadHandle()},
		db:   db,
	}
}

func (w *WHandle) Close() {
	w.db.Close()

	var r RHandle
	w.Read = r
}

func (r *RHandle) Close() {
	r.db.Close()
}

func (m *ProcessManager) FindById(id ProcessId) (*Process, error) {
	r := m.ReadHandle()
	defer r.Close()

	return r.FindById(id)
}

func (m *ProcessManager) IsAlive(id ProcessId) bool {
	r := m.ReadHandle()
	defer r.Close()

	return r.IsAlive(id)
}

func (m *ProcessManager) CopyOutData() []Process {
	r := m.ReadHandle()
	defer r.Close()

	return r.CopyOutData()
}

func (m *ProcessManager) Remove(id ProcessId) error {
	w := m.WriteHandle()
	defer w.Close()

	return w.Remove(id)
}

func (m *ProcessManager) Kill(id ProcessId) error {
	w := m.WriteHandle()
	defer w.Close()

	return w.Kill(id)
}

func (m *ProcessManager) SpawnFromPathVar(config ProcessConfig) (*Process, error) {
	w := m.WriteHandle()
	defer w.Close()

	return w.SpawnFromPathVar(config)
}

func (m *ProcessManager) Spawn(config ProcessConfig) (*Process, error) {
	w := m.WriteHandle()
	defer w.Close()

	return w.Spawn(config)
}
