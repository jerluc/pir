package pir

import (
	"fmt"
	"net"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	// The default multicast broadcast address
	DefaultBroadcastIP = "224.0.0.1"
	// The default broadcast interval
	DefaultBroadcastInterval = 1 * time.Second
)

// A Group represents the current host's logical representation of a LAN-based
// peer network, including keeping track of peer membership.
type Group struct {
	// The group's textual name
	Name string
	// The multicast broadcast address to use for membership broadcasts
	BroadcastAddress *net.UDPAddr
	// The broadcast interval between broadcasts
	BroadcastInterval time.Duration
	// A mapping of peer UUID4s to their corresponding trackers
	peerTrackers map[string]*Tracker
	// A lock on peerTrackers
	peerMutex sync.Mutex
	// A list of MembershipListener who are notified of group membership events
	membershipListeners []MembershipListener
}

// NewGroup creates a new group with the given name, listening on the given
// multicast broadcast port
func NewGroup(name string, port int) *Group {
	fullAddr := fmt.Sprintf("%s:%d", DefaultBroadcastIP, port)
	resolvedAddr, resolveErr := net.ResolveUDPAddr("udp", fullAddr)
	if resolveErr != nil {
		log.Fatal(resolveErr)
	}

	return &Group{
		Name:                name,
		BroadcastAddress:    resolvedAddr,
		BroadcastInterval:   DefaultBroadcastInterval,
		peerTrackers:        make(map[string]*Tracker, 0),
		membershipListeners: make([]MembershipListener, 0),
	}
}

// Adds a MembershipListener to the list of listeners to be notified of group
// membership events
func (g *Group) AddListener(listener MembershipListener) {
	g.peerMutex.Lock()
	defer g.peerMutex.Unlock()

	g.membershipListeners = append(g.membershipListeners, listener)
}

// Notifies each MembershipListener of the MembershipEvent that occurred. If a
// MembershipListener returns true, the listener is retained. Otherwise the
// listener is unregistered and will no longer receive future group membership
// updates.
func (g *Group) notifyListeners(peer *Peer, eventType MembershipEventType) {
	retainedListeners := make([]MembershipListener, 0)
	for _, listener := range g.membershipListeners {
		keep := listener(MembershipEvent{g, peer, eventType})
		if keep {
			retainedListeners = append(retainedListeners, listener)
		}
	}
	g.membershipListeners = retainedListeners
}

// AddPeer adds a peer to the group's membership list, if it doesn't already
// exist. To add the peer, a new tracker is first  created for the peer and
// started, then the membershipListeners are notified of the PeerAdded event.
func (g *Group) AddPeer(peer *Peer) {
	g.peerMutex.Lock()
	defer g.peerMutex.Unlock()

	if _, exists := g.peerTrackers[peer.ID]; !exists {
		tracker := NewTracker(peer, NewHealthChecker(peer), g.RemovePeer)
		go tracker.Track()
		log.Infof("Adding peer tracker [ %s ]", tracker)
		g.peerTrackers[peer.ID] = tracker
		g.notifyListeners(peer, PeerAdded)
	}
}

// RemovePeer removes a peer from the group's membership list, if it exists, and
// notifies each of the membershipListeners of the PeerRemoved event. Note that
// since this typically occurs in response to peer death, the call to kill the
// corresponding peer tracker will normally be a no-op. However, in the case
// that RemovePeer is called from an outside client, the call to kill the peer
// tracker will have side-effects.
func (g *Group) RemovePeer(peer *Peer) {
	g.peerMutex.Lock()
	defer g.peerMutex.Unlock()

	if tracker, exists := g.peerTrackers[peer.ID]; exists {
		log.Warnf("Removing peer tracker [ %s ]", g.peerTrackers[peer.ID])
		tracker.Kill()
		delete(g.peerTrackers, peer.ID)
		g.notifyListeners(peer, PeerRemoved)
	}
}

func (g *Group) String() string {
	return fmt.Sprintf("Group{ name: %s, broadcast: %s }", g.Name, g.BroadcastAddress)
}
