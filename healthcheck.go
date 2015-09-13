package pir

import (
  "net"

  log "github.com/golang/glog"
)

const (
  HEALTHCHECK_BUFFER_SIZE = 1
  HEALTHCHECK_PAYLOAD     = byte(0)
)

func getHealthCheckAddr() (net.Addr, error) {
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

func StartHealthCheckServer(healthCheckAddr chan net.Addr) (error) {
  addr, err := getHealthCheckAddr()
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
  fill, err := conn.Write([]byte{HEALTHCHECK_PAYLOAD})
  if fill != 1 || err != nil {
    return false
  }

  return true
}

func readHealth(conn net.Conn) bool {
  buffer := make([]byte, HEALTHCHECK_BUFFER_SIZE)
  fill, err := conn.Read(buffer)
  if fill != 1 || err != nil || buffer[0] != HEALTHCHECK_PAYLOAD {
    return false
  }

  return true
}

func NewHealthCheck(peer *Peer) func() bool {
  return func() bool {
    conn, err := net.DialTCP("tcp", nil, peer.healthCheck.(*net.TCPAddr))
    if err != nil {
      return false
    }

    defer conn.Close()

    return sendHealth(conn) && readHealth(conn)
  }
}
