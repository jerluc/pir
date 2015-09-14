package pir

import (
	"fmt"
)

const (
	// PeerAdded occurs when a new peer is added to a group
	PeerAdded MembershipEventType = "peer-added"
	// PeerRemoved occurs when an existing peer is removed from a group
	PeerRemoved = "peer-removed"
)

// A kind of membership event
type MembershipEventType string

// A MembershipEvent contains metadata about a change in group membership
type MembershipEvent struct {
	// The group in which the membership change occurred
	Group *Group
	// The peer that changed
	Peer *Peer
	// The kind of membership change that occurred
	Type MembershipEventType
}

func (m MembershipEvent) String() string {
	return fmt.Sprintf("MembershipEvent{ group: %s, peer: %s, type: %s }", m.Group, m.Peer, m.Type)
}

// A MembershipListener is a function which is fired when a membership change
// occurs in a given group. To continue receiving future updates, the
// MembershipListener should return true; returning false indicates that this
// listener no longer wishes to receive updates.
type MembershipListener func(MembershipEvent) bool
