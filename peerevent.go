package pir

import (
  "fmt"
)

const (
  PeerAdded   PeerEventType = "peer-added"
  PeerRemoved               = "peer-removed"
)

type PeerEventType string

type PeerEvent struct {
  Peer  *Peer
  Event PeerEventType
}

func (p PeerEvent) String() string {
  return fmt.Sprintf("PeerEvent{ peer: %s, event: %s }", p.Peer, p.Event)
}

type PeerListener func(PeerEvent) bool
