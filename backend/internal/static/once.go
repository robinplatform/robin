package static

import "sync"

func CreateOnce[T any](creator func() T) func() T {
	var value T
	var once sync.Once

	return func() T {
		once.Do(func() {
			value = creator()
		})

		return value
	}
}
