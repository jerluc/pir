package pir

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

const (
	// Buffer size to use for UDP broadcasts (read/write)
	UDPBufferSize = 8192
)

// A Peer is a logical representation of a peering device participating in a
// group. This same representation is used both for remote and local peers, for
// simplicity and consistency.
type Peer struct {
	// UUID4 to identify this peer uniquely; should be unique even across multiple
	// groups. If this ID is stable/reproducible, group membership will also be
	// stable/reproducible.
	ID string
	// The communications URI specification; Pir does not interpret the meaning of
	// this URI itself, as this URI is application/service specific
	CommsSpec *url.URL
	// The healthcheck URI specification; this URI is used by Pir to maintain
	// logical group membership, as viewed by the local Peer instance
	HealthCheckSpec *url.URL
}

func generateId() string {
	return uuid.NewV4().String()
}

// NewPeer creates a new peer from the provided communications and healthcheck
// URIs. If either URI is invalid, an error is returned.
func NewPeer(commsSpecStr string, healthCheckSpecStr string) (*Peer, error) {
	commsSpec, err := url.Parse(commsSpecStr)
	if err != nil {
		return nil, err
	}

	healthCheckSpec, err := url.Parse(healthCheckSpecStr)
	if err != nil {
		return nil, err
	}

	return &Peer{generateId(), commsSpec, healthCheckSpec}, nil
}

// Join joins the given group. Internally, this operates the same as both
// advertising and subscribing to the same group
func (p *Peer) Join(group *Group) {
	go p.Advertise(group)
	go p.Subscribe(group)
}

func sendAdvertisement(group *Group, peer *Peer, conn net.Conn) {
	payload := fmt.Sprintf("%s|%s|%s|%s", group.Name, peer.ID, peer.HealthCheckSpec, peer.CommsSpec)
	conn.Write([]byte(payload))
}

// Advertise advertises this peer on the given group during each broadcast
// interval using the group's broadcast address. The advertising packet is
// formatted as: `GROUP_NAME|UUID4|proto://x.x.x.x:xxxxx|proto://x.x.x.x:xxxxx`
func (p *Peer) Advertise(group *Group) {
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

// An InvalidPeerBroadcastErr occurs when a broadcast payload cannot be
// interpreted. This typically isn't fatal, but can be a sign of a broken or
// outdated protocol implementation.
type InvalidPeerBroadcastErr struct {
	payload string
}

func (i InvalidPeerBroadcastErr) Error() string {
	return fmt.Sprintf("Received invalid peer broadcast: %s", i.payload)
}

func (p *Peer) parsePeerBroadcast(group *Group, payload string) (*Peer, bool, error) {
	parts := strings.Split(payload, "|")
	if len(parts) != 4 {
		return nil, false, &InvalidPeerBroadcastErr{payload}
	}

	if groupName := parts[0]; groupName != group.Name {
		return nil, false, nil
	}

	id := parts[1]

	healthCheckSpec, err := url.Parse(parts[2])
	if err != nil {
		return nil, true, err
	}

	commsSpec, err := url.Parse(parts[3])
	if err != nil {
		return nil, true, err
	}

	return &Peer{id, commsSpec, healthCheckSpec}, true, nil
}

// Subscribe subscribes a peer to a group. By subscribing to a group, a peer
// receives global updates of group membership across the network.
func (p *Peer) Subscribe(group *Group) {
	groupConn, err := net.ListenMulticastUDP("udp", nil, group.BroadcastAddress)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Subscribed to group [ %s ]", group)
	for {
		buffer := make([]byte, UDPBufferSize)
		fill, _, err := groupConn.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}

		payload := string(buffer[:fill])

		peer, belongsToGroup, parseErr := p.parsePeerBroadcast(group, payload)
		if parseErr != nil {
			log.Warn("Could not handle broadcast:", parseErr)
		}

		if belongsToGroup && peer.ID != p.ID {
			group.AddPeer(peer)
		}
	}
}

func (p *Peer) String() string {
	return fmt.Sprintf("Peer{ id: %s, commsSpec: %s, healthCheckSpec: %s }",
		p.ID, p.CommsSpec, p.HealthCheckSpec)
}
