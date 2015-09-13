package pir

import (
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	HealthCheckBufferSize = 1
	HealthCheckPayload    = byte(0)
)

type HealthCheck struct {
	addr net.Addr
}

func NewHealthCheck() *HealthCheck {
	return &HealthCheck{nil}
}

func (h *HealthCheck) Start() error {
	addr := make(chan net.Addr, 1)
	go listen(addr)
	select {
	case healthCheckAddr := <-addr:
		h.addr = healthCheckAddr
		return nil
	case <-time.After(HealthCheckListenTimeout):
		return &HealthCheckListenTimeoutErr{}
	}
}

func generateHealthCheckAddr() (net.Addr, error) {
	localIP, err := GetLocalIP()
	if err != nil {
		return nil, err
	}

	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(localIP, "0"))
	if err != nil {
		return nil, err
	}

	return addr, nil
}

func listen(healthCheckAddr chan net.Addr) error {
	addr, err := generateHealthCheckAddr()
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", addr.(*net.TCPAddr))
	if err != nil {
		return err
	}

	healthCheckAddr <- listener.Addr()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			defer conn.Close()
			if !readHealth(conn) || !sendHealth(conn) {
				// Connection is dead with client
				return
			}
		}()
	}

	return nil
}

func sendHealth(conn net.Conn) bool {
	fill, err := conn.Write([]byte{HealthCheckPayload})
	if fill != 1 || err != nil {
		return false
	}

	return true
}

func readHealth(conn net.Conn) bool {
	buffer := make([]byte, HealthCheckBufferSize)
	fill, err := conn.Read(buffer)
	if fill != 1 || err != nil || buffer[0] != HealthCheckPayload {
		return false
	}

	return true
}

func (h *HealthCheck) URISpec() string {
	return fmt.Sprintf("tcp://%s", h.addr)
}

func NewHealthChecker(peer *Peer) func() bool {
	return func() bool {
		healthCheckAddr, err := ResolveURISpec(peer.healthCheckSpec.String())
		if err != nil {
			return false
		}

		conn, err := net.DialTCP("tcp", nil, healthCheckAddr.(*net.TCPAddr))
		if err != nil {
			return false
		}

		defer conn.Close()

		return sendHealth(conn) && readHealth(conn)
	}
}
