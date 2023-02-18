package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type HttpHealthCheck struct {
	Method string
	Url    string
}

func CheckHttp(healthCheck HttpHealthCheck) bool {
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
	cancel()

	// Non-200 status codes are fine, because we still got a response from an
	// http server
	return true
}

type TcpHealthCheck struct {
	ipv4 bool
	port int
}

func CheckTcp(healthCheck TcpHealthCheck) bool {
	host := "::1"
	if healthCheck.ipv4 {
		host = "127.0.01"
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, healthCheck.port))
	if err != nil {
		return false
	}
	conn.Close()

	return true
}
