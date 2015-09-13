package pir

import (
  "fmt"
  "net"
  "sync"
  "time"

  log "github.com/golang/glog"
)

const (
	DEFAULT_BROADCAST_IP       = "224.0.0.1"
  DEFAULT_BROADCAST_INTERVAL = 1 * time.Second
)

type Group struct {
  Name              string
  BroadcastAddress  *net.UDPAddr
  BroadcastInterval time.Duration
  peerTrackers      map[string]*Tracker
  peerMutex         sync.Mutex
  peerListeners     []PeerListener
}

func NewGroup(name string, port int) *Group {
  fullAddr := fmt.Sprintf("%s:%d", DEFAULT_BROADCAST_IP, port)
  resolvedAddr, resolveErr := net.ResolveUDPAddr("udp", fullAddr)
  if resolveErr != nil {
    // TODO: How to best do this?
    log.Fatal(resolveErr)
  }

  return &Group{name, resolvedAddr, DEFAULT_BROADCAST_INTERVAL,
    make(map[string]*Tracker, 0), sync.Mutex{}, make([]PeerListener, 0)}
}

func (g *Group) AddListener(listener PeerListener) {
  g.peerMutex.Lock()
  defer g.peerMutex.Unlock()

  g.peerListeners = append(g.peerListeners, listener)
}

func (g *Group) notifyListeners(peer *Peer, eventType PeerEventType) {
  retainedListeners := make([]PeerListener, 0)
  for _, listener := range g.peerListeners {
    keep := listener(PeerEvent{peer, eventType})
    if keep {
      retainedListeners = append(retainedListeners, listener)
    }
  }
  g.peerListeners = retainedListeners
}

func (g *Group) AddPeer(peer *Peer) {
  g.peerMutex.Lock()
  defer g.peerMutex.Unlock()

  if _, exists := g.peerTrackers[peer.ID]; !exists {
    tracker := NewTracker(peer, NewHealthCheck(peer), g.RemovePeer)
    log.Infof("Adding peer tracker [ %s ]", tracker)
    g.peerTrackers[peer.ID] = tracker
    g.notifyListeners(peer, PeerAdded)
  }
}

func (g *Group) RemovePeer(peer *Peer) {
  g.peerMutex.Lock()
  defer g.peerMutex.Unlock()

  log.Infof("Removing peer tracker [ %s ]", g.peerTrackers[peer.ID])
  delete(g.peerTrackers, peer.ID)
  g.notifyListeners(peer, PeerRemoved)
}

func (g *Group) String() string {
  return fmt.Sprintf("Group{ name: %s, broadcast: %s }", g.Name, g.BroadcastAddress)
}
