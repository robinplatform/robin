package process

import "robinplatform.dev/internal/model"

type RHandle struct {
	db model.RHandle[Process]
}

type WHandle struct {
	db model.WHandle[Process]
}

func (manager *ProcessManager) ReadHandle() RHandle {
	return RHandle{db: manager.db.ReadHandle()}
}

func (manager *ProcessManager) WriteHandle() WHandle {
	return WHandle{db: manager.db.WriteHandle()}
}

func (w *WHandle) Close() {
	w.db.Close()
}

func (r *RHandle) Close() {
	r.db.Close()
}

func (m *ProcessManager) FindById(id ProcessId) (*Process, error) {
	r := m.ReadHandle()
	defer r.Close()

	return r.FindById(id)
}

func (w *WHandle) FindById(id ProcessId) (*Process, error) {
	r := RHandle{db: w.db.UncloseableReadHandle()}
	return r.FindById(id)
}

func (m *ProcessManager) IsAlive(id ProcessId) bool {
	r := m.ReadHandle()
	defer r.Close()

	return r.IsAlive(id)
}

func (w *WHandle) IsAlive(id ProcessId) bool {
	r := RHandle{db: w.db.UncloseableReadHandle()}
	return r.IsAlive(id)
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

func (m *ProcessManager) SpawnPath(config ProcessConfig) (*Process, error) {
	w := m.WriteHandle()
	defer w.Close()

	return w.SpawnPath(config)
}

func (m *ProcessManager) Spawn(config ProcessConfig) (*Process, error) {
	w := m.WriteHandle()
	defer w.Close()

	return w.Spawn(config)
}
