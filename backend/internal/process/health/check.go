package health

import (
	"encoding/json"
	"fmt"

	"robinplatform.dev/internal/log"
)

var logger = log.New("process.health")

type RunningProcessInfo struct {
	Pid int
}

type HealthCheck interface {
	Check(p RunningProcessInfo) bool
}

type SerializableHealthCheck struct {
	Type  string      `json:"type"`
	Check HealthCheck `json:"check"`
}

func GetTypeFromHealthCheck(check HealthCheck) (string, error) {
	switch check.(type) {
	case ProcessHealthCheck:
		return "process", nil

	case TcpHealthCheck:
		return "tcp", nil

	case HttpHealthCheck:
		return "http", nil

	default:
		return "", fmt.Errorf("did not recognize healthcheck type")
	}
}

func (check *SerializableHealthCheck) UnmarshalJSON(data []byte) error {
	obj := struct {
		Type  string          `json:"type"`
		Check json.RawMessage `json:"check"`
	}{}

	err := json.Unmarshal(data, &obj)
	if err != nil {
		return err
	}

	check.Type = obj.Type

	switch obj.Type {
	case "process":
		check.Check = &ProcessHealthCheck{}

	case "http":
		check.Check = &HttpHealthCheck{}

	case "tcp":
		check.Check = &TcpHealthCheck{}

	default:
		if obj.Type == "" {
			return fmt.Errorf("health check didn't have a type")
		}

		return fmt.Errorf("found unrecognized health check type: '%s'", obj.Type)
	}

	err = json.Unmarshal(data, check.Check)
	if err != nil {
		return err
	}

	return nil
}
