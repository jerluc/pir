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
	HealthCheckListenTimeout = 2 * time.Second
	UDPBufferSize            = 8192
)

type Peer struct {
	ID              string
	commsSpec       *url.URL
	healthCheckSpec *url.URL
}

func generateId() string {
	return uuid.NewV4().String()
}

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

func (p *Peer) Join(group *Group) {
	go p.Advertise(group)
	go p.Subscribe(group)
}

type HealthCheckListenTimeoutErr struct{}

func (h HealthCheckListenTimeoutErr) Error() string {
	return "Failed to start healthcheck server"
}

func sendAdvertisement(group *Group, peer *Peer, conn net.Conn) {
	payload := fmt.Sprintf("%s|%s|%s|%s", group.Name, peer.ID, peer.healthCheckSpec, peer.commsSpec)
	conn.Write([]byte(payload))
}

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

func (p *Peer) CommsSpec() *url.URL {
	return p.commsSpec
}

func (p *Peer) String() string {
	return fmt.Sprintf("Peer{ id: %s, commsSpec: %s, healthCheckSpec: %s }", p.ID, p.commsSpec, p.healthCheckSpec)
}
