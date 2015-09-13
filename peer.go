package pir

import (
  "fmt"
  "net"
  "strings"
  "time"

  log "github.com/golang/glog"
  uuid "github.com/satori/go.uuid"
)

const (
  HEALTHCHECK_LISTEN_TIMEOUT = 2 * time.Second
  DG_BUFFER_SIZE             = 8192
)

type Peer struct {
  ID              string
  healthCheck     net.Addr
}

func generateId() string {
  return uuid.NewV4().String()
}

func NewPeer() *Peer {
  return &Peer{generateId(), nil}
}

func (p *Peer) Join(group *Group) {
  go p.advertise(group)
  go p.subscribe(group)
}

type HealthCheckListenTimeoutErr struct{}
func (h HealthCheckListenTimeoutErr) Error() string {
  return "Failed to start healthcheck server"
}

func (p *Peer) listen() error {
  addr := make(chan net.Addr, 1)
  go StartHealthCheckServer(addr)
  select {
  case healthCheckAddr := <-addr:
    p.healthCheck = healthCheckAddr
    return nil
  case <-time.After(HEALTHCHECK_LISTEN_TIMEOUT):
    return &HealthCheckListenTimeoutErr{}
  }
}

func sendAdvertisement(group *Group, peer *Peer, conn net.Conn) {
  payload := fmt.Sprintf("%s|%s|%s", group.Name, peer.ID, peer.healthCheck)
  conn.Write([]byte(payload))
}

func (p *Peer) advertise(group *Group) {
  if err := p.listen(); err != nil {
    log.Fatal(err)
  }

	groupConn, err := net.DialUDP("udp", nil, group.BroadcastAddress)
  if err != nil {
    log.Fatal(err)
  }

  log.Infof("Advertising on group [ %s ]", group)
	for {
    sendAdvertisement(group, p, groupConn)
		time.Sleep(group.BroadcastInterval)
	}
}

type InvalidPeerBroadcastErr struct{
  payload string
}

func (i InvalidPeerBroadcastErr) Error() string {
  return fmt.Sprintf("Received invalid peer broadcast: %s", i.payload)
}

func (p *Peer) parsePeerBroadcast(group *Group, payload string) (*Peer, bool, error) {
  parts := strings.Split(payload, "|")
  if len(parts) != 3 {
    return nil, false, &InvalidPeerBroadcastErr{payload}
  }

  if groupName := parts[0]; groupName != group.Name {
    return nil, false, nil
  }

  id := parts[1]

  healthCheck, err := net.ResolveTCPAddr("tcp", parts[1])
  if err != nil {
    return nil, true, err
  }

  return &Peer{id, healthCheck}, true, nil
}

func (p *Peer) subscribe(group *Group) {
  groupConn, err := net.ListenMulticastUDP("udp", nil, group.BroadcastAddress)
  if err != nil {
    log.Fatal(err)
  }

  groupConn.SetReadBuffer(DG_BUFFER_SIZE)

  for {
    buffer := make([]byte, DG_BUFFER_SIZE)
    fill, _, err := groupConn.ReadFromUDP(buffer)
    if err != nil {
      log.Fatal(err)
    }

    payload := string(buffer[:fill])

    peer, belongsToGroup, parseErr := p.parsePeerBroadcast(group, payload)
    if parseErr != nil {
      log.Warning("Could not handle broadcast:", parseErr)
    }

    if belongsToGroup && peer.ID != p.ID {
      group.AddPeer(peer)
    }
  }
}

func (p *Peer) String() string {
  return fmt.Sprintf("Peer{ id: %s }", p.ID)
}
