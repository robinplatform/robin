package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type HttpHealthCheck struct {
	Method string `json:"method"`
	Url    string `json:"url"`
}

func (healthCheck HttpHealthCheck) Check(info RunningProcessInfo) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, healthCheck.Method, healthCheck.Url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()

	// We _must_ get a 200 OK response. If the service is designed to return anything
	// else at this route, it should use a TCP health check instead.
	return resp.StatusCode == http.StatusOK
}

type TcpHealthCheck struct {
	IPv4 bool `json:"ipv4"`
	Port int  `json:"port"`
}

func (healthCheck TcpHealthCheck) Check(info RunningProcessInfo) bool {
	host := "::1"
	if healthCheck.IPv4 {
		host = "127.0.01"
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, healthCheck.Port))
	if err != nil {
		return false
	}
	conn.Close()

	return true
}
