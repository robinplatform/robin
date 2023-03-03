package static

import "sync"

type OnceInit[T any] struct {
	value T
	err   error
	once  sync.Once

	Init     func(func() (T, error)) (bool, T, error)
	GetValue func() (T, error)
}

func CreateOnce[T any](creator func() (T, error)) *OnceInit[T] {
	out := &OnceInit[T]{}

	out.GetValue = func() (T, error) {
		out.once.Do(func() {
			out.value, out.err = creator()
		})

		return out.value, out.err
	}

	out.Init = func(creator func() (T, error)) (bool, T, error) {
		didInit := false
		out.once.Do(func() {
			out.value, out.err = creator()
			didInit = true
		})

		if !didInit {
			return false, out.value, out.err
		}

		return true, out.value, out.err
	}

	return out
}
