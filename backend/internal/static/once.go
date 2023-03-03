package static

import "sync"

func CreateOnce[T any](creator func() (T, error)) func() (T, error) {
	var value T
	var err error
	var once sync.Once

	return func() (T, error) {
		once.Do(func() {
			value, err = creator()
		})

		return value, err
	}
}
