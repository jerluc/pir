Pir
===

Pir is a simple library for facilitating in device discovery on a LAN.

### Discovery protocol
Presently, this is done using a fairly common discovery protocol:

* New peers join a group by connecting to a multicast UDP broadcast address.
  * Once joined, each peer broadcasts a simple payload consisting of the participating group, peer ID and healthcheck address: `GROUP_NAME|UUID4|x.x.x.x:xxxxx`
* Each participating peer in the group consumes these broadcasts, creating and updating peer trackers. Trackers are updated by attempting to send and receive data from the healthcheck address.
  * A successful attempt improves tracker health
  * A failure or timeout degrades tracker health
* When tracker health degrades beyond a certain threshold, the tracker is removed, rendering the tracked peer invisible to the tracker

### Example usage
Basic usage:
```go
group := pir.NewGroup("GROUP_NAME", 9999)
peer := pir.NewPeer()
go peer.Join(group)
```
