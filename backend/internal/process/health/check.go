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
	checkType string
	check     HealthCheck
}

func (check SerializableHealthCheck) Check(info RunningProcessInfo) bool {
	return check.check.Check(info)
}

func NewHealthCheck(check HealthCheck) (SerializableHealthCheck, error) {
	switch v := check.(type) {
	case SerializableHealthCheck:
		return v, nil

	case ProcessHealthCheck:
		return SerializableHealthCheck{
			checkType: "process",
			check:     check,
		}, nil

	case TcpHealthCheck:
		return SerializableHealthCheck{
			checkType: "tcp",
			check:     check,
		}, nil

	case HttpHealthCheck:
		return SerializableHealthCheck{
			checkType: "http",
			check:     check,
		}, nil

	default:
		return SerializableHealthCheck{}, fmt.Errorf("did not recognize healthcheck type")
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

	check.checkType = obj.Type

	switch obj.Type {
	case "process":
		c := ProcessHealthCheck{}
		err = json.Unmarshal(data, &c)
		check.check = c

	case "http":
		c := HttpHealthCheck{}
		err = json.Unmarshal(data, &c)
		check.check = c

	case "tcp":
		c := TcpHealthCheck{}
		err = json.Unmarshal(data, &c)
		check.check = c

	default:
		if obj.Type == "" {
			return fmt.Errorf("health check didn't have a type")
		}

		return fmt.Errorf("found unrecognized health check type: '%s'", obj.Type)
	}

	return err

}

func (check *SerializableHealthCheck) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  check.checkType,
		"check": check.check,
	})
}
