Pir
===
[![GoDoc](https://godoc.org/github.com/jerluc/pir?status.svg)](https://godoc.org/github.com/jerluc/pir) [![Build Status](https://travis-ci.org/jerluc/pir.svg)](https://travis-ci.org/jerluc/pir)

Pir is a simple library for facilitating in device discovery on a LAN.

### Installation

##### Using `go get`

```bash
go get github.com/jerluc/pir
```

##### From source

```bash
git clone https://github.com/jerluc/pir.git
cd pir
go install
```

### Example usage

##### Basic usage ([full source](examples/basic))
```go
// Start a healthcheck server
healthcheck := pir.NewHealthCheck()
healthcheck.Start()

// Create a new peer
peer, _ := pir.NewPeer("tcp://10.1.1.1:80", healthcheck.URISpec())

// Join the group on port 9999
group := pir.NewGroup(group, 9999)
peer.Join(group)
```

##### Receiving membership updates
```go
group.AddListener(func(event pir.MembershipEvent) bool {
  fmt.Println("Membership change event has occurred:", event)
  return true
})
```

### Discovery protocol
Presently, this is done using a fairly common discovery protocol:

* New peers join a group by connecting to a multicast UDP broadcast address.
  * Once joined, each peer broadcasts a simple payload consisting of the participating group, peer ID, healthcheck URI, and communications URI: `GROUP_NAME|UUID4|proto://x.x.x.x:xxxxx|proto://x.x.x.x:xxxxx`
* Each participating peer in the group consumes these broadcasts, creating and updating peer trackers. Trackers are updated by attempting to send and receive data over the healthcheck URI:
  * Each successful send+receive improves tracker health
  * Each failure or timeout degrades tracker health
* When tracker health degrades beyond a certain threshold, the tracker is removed, rendering the tracked peer invisible to the tracker
