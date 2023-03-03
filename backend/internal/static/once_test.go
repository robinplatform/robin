package static

import "testing"

func TestOnce(t *testing.T) {
	once := CreateOnce(func() (int, error) {
		return 1, nil
	})

	value, err := once.GetValue()

	if err != nil {
		t.Error(err)
	}
	if value != 1 {
		t.Errorf("Expected 1, got %d", value)
	}

	didInit, value, err := once.Init(func() (int, error) {
		return 2, nil
	})

	if didInit {
		t.Errorf("Should not have initialized twice")
	}
	if err != nil {
		t.Error(err)
	}
	if value != 1 {
		t.Errorf("Expected 2, got %d", value)
	}

}
