package pir

import (
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
)

// TODO: Make this mechanism suck less; maybe use TCP keep-alive and monitor
// connection status rather than actual making a request/response protocol?

const (
	// The timeout duration after attempting to create a healthcheck server after
	// which to fail
	HealthCheckListenTimeout = 2 * time.Second
	// The TCP buffer size for healthcheck reads/writes
	HealthCheckBufferSize = 1
	// The payload to send and receive from the healthcheck
	HealthCheckPayload = byte(0)
)

// A HealthCheck acts as a server to handle the health-checking protocol
type HealthCheck struct {
	// The server address
	addr net.Addr
}

// A HealthCheckListenTimeoutErr occurs when the healthcheck server fails to
// start after the HealthCheckListenTimeout elapses
type HealthCheckListenTimeoutErr struct{}

func (h HealthCheckListenTimeoutErr) Error() string {
	return "Failed to start healthcheck server"
}

// Creates a new healthcheck server
func NewHealthCheck() *HealthCheck {
	return &HealthCheck{nil}
}

// Start asynchronously starts listening on the next available port, handling
// the builtin healthcheck protocol.
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

// URISpec is a convenience method to construct the URI specification for this
// healthcheck server
func (h *HealthCheck) URISpec() string {
	return fmt.Sprintf("tcp://%s", h.addr)
}

// NewHealthChecker acts as a healthchecking function for a Tracker, by using
// the builtin healthchecking protocol to query for peer health.
func NewHealthChecker(peer *Peer) func() bool {
	return func() bool {
		healthCheckAddr, err := ResolveURISpec(peer.HealthCheckSpec.String())
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
