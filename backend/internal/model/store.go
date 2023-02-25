package model

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Store[Model any] struct {
	// FilePath is the path to the json file where the data should be stored.
	FilePath string

	data  []Model
	rwMux sync.RWMutex
}

type WHandle[Model any] struct {
	store *Store[Model]
}

type RHandle[Model any] struct {
	store *Store[Model]
}

func (store *Store[Model]) Open() error {
	store.rwMux = sync.RWMutex{}
	store.rwMux.Lock()
	defer store.rwMux.Unlock()

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

func (store *Store[Model]) WriteHandle() WHandle[Model] {
	store.rwMux.Lock()
	return WHandle[Model]{store}
}

func (store *Store[Model]) ReadHandle() RHandle[Model] {
	store.rwMux.RLock()
	return RHandle[Model]{store}
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
	// I am not used to this language enough to feel OK with this
	// code by default, but it's recommended by Go Wiki and also
	// in principle it makes sense:
	// https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating

	data := w.store.data
	out := data[:0]
	for _, row := range data {
		if !matcher(row) {
			// Out uses the same backing storage because it was created
			// from the original sice, so there's no additional allocation here
			out = append(out, row)
		}
	}

	// Clear out previous items to ensure garbage collection
	var zero Model
	for i := len(out); i < len(data); i++ {
		data[i] = zero // or the zero value of T
	}

	w.store.data = out

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
