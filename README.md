Pir
===
[![GoDoc](https://godoc.org/github.com/jerluc/pir?status.svg)](https://godoc.org/github.com/jerluc/pir) [![Build Status](https://travis-ci.org/jerluc/pir.svg)](https://travis-ci.org/jerluc/pir)

Pir is a simple library for facilitating in device discovery on a LAN.

### Discovery protocol
Presently, this is done using a fairly common discovery protocol:

* New peers join a group by connecting to a multicast UDP broadcast address.
  * Once joined, each peer broadcasts a simple payload consisting of the participating group, peer ID, healthcheck address, and communications URI: `GROUP_NAME|UUID4|x.x.x.x:xxxxx|proto://x.x.x.x:xxxxx`
* Each participating peer in the group consumes these broadcasts, creating and updating peer trackers. Trackers are updated by attempting to send and receive data over TCP through the healthcheck address:
  * Each successful send+receive improves tracker health
  * Each failure or timeout degrades tracker health
* When tracker health degrades beyond a certain threshold, the tracker is removed, rendering the tracked peer invisible to the tracker

### Example usage
Basic usage:
```go
// Create a peer who communicates over TCP on 10.1.1.1:12000
peer := pir.NewPeer("tcp://10.1.1.1:12000")
// Create the `abc` group on port 9999
group := pir.NewGroup("abc", 9999)
// Join the `abc` group
go peer.Join(group)
```

Receiving membership updates:
```go
// Create a peer who communicates over TCP on 10.1.1.1:12000
peer := pir.NewPeer("tcp://10.1.1.1:12000")
// Create the `abc` group on port 9999
group := pir.NewGroup("abc", 9999)
// Register membership listener
group.AddListener(func(event pir.PeerEvent) bool {
  fmt.Println("Membership change event has occurred:", event)
  return true
})
// Join the `abc` group
go peer.Join(group)
```
