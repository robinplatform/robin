package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store of information that gets loaded from disk on initialization and persisted to disk on write.
// It provides the following promises:
// - Modifications like insertions or mutations are persisted
// - Data is persisted using JSON
// - Data is ONLY read on initialization
type Store[Model any] struct {
	// FilePath is the path to the json file where the data should be stored.
	FilePath string

	data  []Model
	rwMux *sync.RWMutex
}

type WHandle[Model any] struct {
	store *Store[Model]
}

type RHandle[Model any] struct {
	store *Store[Model]
}

func (store *Store[Model]) open() error {
	store.rwMux.Lock()
	defer store.rwMux.Unlock()

	if err := os.MkdirAll(filepath.Dir(store.FilePath), 0755); err != nil {
		return fmt.Errorf("failed to create datastore directory: %w", err)
	}

	buf, err := os.ReadFile(store.FilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open datastore: %w", err)
	}

	if err := json.Unmarshal(buf, &store.data); err != nil {
		return fmt.Errorf("failed to unmarshal datastore: %w", err)
	}

	return nil
}

func NewStore[Model any](dbPath string) (Store[Model], error) {
	store := Store[Model]{
		FilePath: dbPath,
		rwMux:    &sync.RWMutex{},
	}
	err := store.open()
	return store, err
}

func (store *Store[Model]) WriteHandle() WHandle[Model] {
	store.rwMux.Lock()
	return WHandle[Model]{store}
}

func (store *Store[Model]) ReadHandle() RHandle[Model] {
	store.rwMux.RLock()
	return RHandle[Model]{store}
}

// Creates a read handle. Closing this will cause a failure in the underlying RWLock,
// because it's currently locked for writing, but read handles only close for
// reading.
func (w *WHandle[Model]) UncloseableReadHandle() RHandle[Model] {
	return RHandle[Model]{store: w.store}
}

func (w *WHandle[Model]) Close() {
	w.store.rwMux.Unlock()
}

func (r *RHandle[Model]) Close() {
	r.store.rwMux.RUnlock()
}

func (w *WHandle[Model]) Insert(row Model) error {
	w.store.data = append(w.store.data, row)

	return w.store.flush()
}

func (w *WHandle[Model]) Find(matcher func(row Model) bool) (Model, bool) {
	r := RHandle[Model]{w.store}
	return r.Find(matcher)
}

func (w *WHandle[Model]) Delete(matcher func(row Model) bool) error {
	data := make([]Model, 0, len(w.store.data))
	for _, row := range w.store.data {
		if !matcher(row) {
			data = append(data, row)
		}
	}
	w.store.data = data

	return w.store.flush()
}

func (r *RHandle[Model]) Find(matcher func(row Model) bool) (Model, bool) {
	for _, row := range r.store.data {
		if matcher(row) {
			return row, true
		}
	}

	var zero Model
	return zero, false
}

func (store *Store[Model]) Find(matcher func(row Model) bool) (Model, bool) {
	r := store.ReadHandle()
	defer r.Close()

	return r.Find(matcher)
}

func (r *RHandle[Model]) ShallowCopyOutData() []Model {
	out := make([]Model, 0, len(r.store.data))
	out = append(out, r.store.data...)

	return out
}

func (store *Store[Model]) ShallowCopyOutData() []Model {
	r := store.ReadHandle()
	defer r.Close()

	return r.ShallowCopyOutData()
}

func (w *WHandle[Model]) ForEach(f func(*Model)) {
	for i := 0; i < len(w.store.data); i++ {
		f(&w.store.data[i])
	}

	w.store.flush()
}

func (store *Store[Model]) ForEachWriting(f func(*Model)) {
	// This is called ForEach writing because I wanted to make explicit that it
	// it would take a write lock internally
	w := store.WriteHandle()
	defer w.Close()

	w.ForEach(f)
}

func (store *Store[_]) flush() error {
	buf, err := json.Marshal(store.data)
	if err != nil {
		return fmt.Errorf("failed to marshal datastore: %w", err)
	}

	if err := os.WriteFile(store.FilePath, buf, 0755); err != nil {
		return fmt.Errorf("failed to save datastore: %w", err)
	}

	return nil
}
