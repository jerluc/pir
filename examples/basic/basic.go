package main

import (
	"flag"

	log "github.com/Sirupsen/logrus"
	"github.com/jerluc/pir"
)

// The group that the peer should join
var group string

func init() {
	// -group GROUP_NAME; defaults to `abc`
	flag.StringVar(&group, "group", "abc", "Group to join")
}

func main() {
	// Parse CLI flags
	flag.Parse()

	// Start a healthcheck server
	healthcheck := pir.NewHealthCheck()
	healthcheck.Start()

	// Create a new peer
	peer, _ := pir.NewPeer("tcp://10.1.1.1:80", healthcheck.URISpec())

	// Join the group on port 9999
	group := pir.NewGroup(group, 9999)
	group.AddListener(func(event pir.MembershipEvent) bool {
		log.Info("Membership change event has occurred:", event)
		return true
	})
	peer.Join(group)

	// Wait indefinitely
	select {}
}
