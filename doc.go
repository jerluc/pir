/*
Package pir is a simple library for facilitating device discovery on a LAN.

The simplest way to use Pir is to:

1. Start a healthcheck server
    healthcheck := pir.NewHealthCheck()
    healthcheck.Start()
2. Create a new peer
    peer, _ := pir.NewPeer("tcp://10.1.1.1:80", healthcheck.URISpec())
3. Join a group
    group := pir.NewGroup(group, 9999)
    peer.Join(group)

For the full source, see: https://github.com/jerluc/pir
*/
package pir
