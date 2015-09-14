package pir

import (
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

// TODO: Use some kind of exponential backoff and retention

const (
	// The time interval between healthchecks
	HealthCheckInterval = 2 * time.Second
	// How long to wait on a healthcheck before timing out
	HealthCheckTimeout = 2 * time.Second
	// The failure threshold for reaping a peer
	ReaperThreshold = 3
)

// A Tracker is responsible for keeping track of the health status for a given
// peer.
type Tracker struct {
	// The peer to track
	peer *Peer
	// A function to determine health status for the peer. Returning false
	// degrades the overall health of the peer.
	healthCheck func() bool
	// A function which is called upon peer death as determined by the
	// ReaperThreshold.
	reaper func(*Peer)
	// Holds a read/write lock on the unhealthyCount
	healthMutex sync.RWMutex
	// The number of healthcheck failures, as determinged by either receipt of
	// failure from the healthCheck function, or from timing out
	unhealthyCount int
	// A singly-buffered channel for triggering instant death of a peer
	deathChan chan bool
}

// NewTracker creates a Tracker instance
func NewTracker(peer *Peer, healthCheck func() bool, reaper func(*Peer)) *Tracker {
	return &Tracker{
		peer:        peer,
		healthCheck: healthCheck,
		reaper:      reaper,
		deathChan:   make(chan bool, 1),
	}
}

// MarkHealthy linearly forgives previous healthcheck failures (but cannot)
// repair beyond zero.
func (t *Tracker) MarkHealthy() {
	t.healthMutex.Lock()
	defer t.healthMutex.Unlock()

	if t.unhealthyCount > 0 {
		t.unhealthyCount -= 1
	}
}

// MarkUnhealthy linearly degrades with peer health
func (t *Tracker) MarkUnhealthy() {
	t.healthMutex.Lock()
	defer t.healthMutex.Unlock()

	t.unhealthyCount += 1
}

// Kill triggers instant death of the peer tracker by sending a poison pill onto
// the death channel
func (t *Tracker) Kill() {
	select {
	case <-t.deathChan:
		// Do nothing!
	default:
		t.deathChan <- true
	}
}

// Track starts tracking the peer by calling IsAlive on each HealthCheckInterval.
// When IsAlive finally returns false (or an unexpected panic occurs), the
// reaper function gets automatically triggered.
func (t *Tracker) Track() {
	defer t.reaper(t.peer)
	for t.IsAlive() {
		time.Sleep(HealthCheckInterval)
	}
}

func (t *Tracker) remainingHealth() string {
	t.healthMutex.RLock()
	defer t.healthMutex.RUnlock()

	return fmt.Sprintf("(%d/%d)", ReaperThreshold-t.unhealthyCount, ReaperThreshold)
}

func (t *Tracker) doHealthCheck() <-chan bool {
	healthy := make(chan bool, 1)
	go func() {
		healthy <- t.healthCheck()
	}()
	return healthy
}

// IsAlive queries for peer health. If a value is waiting on the deathChan,
// IsAlive instantly fails. Otherwise, either the healthCheck function returns
// in the given time, determining peer health, or a subsequent timeout is
// surpassed, causing health degradation. Finally, if the unhealthyCount
// surpasses the ReaperThreshold, IsAlive fails.
func (t *Tracker) IsAlive() bool {
	select {
	case <-t.deathChan:
		log.Warnf("Peer is dead [ %s ]", t)
		return false
	case healthy := <-t.doHealthCheck():
		if !healthy {
			t.MarkUnhealthy()
			log.Warnf("Peer is unhealthy [ %s ]", t)
		} else {
			t.MarkHealthy()
			log.Infof("Peer is healthy [ %s ]", t)
		}
	case <-time.After(HealthCheckTimeout):
		t.MarkUnhealthy()
		log.Warnf("Healthcheck timed out for peer [ %s ]", t)
	}

	t.healthMutex.RLock()
	defer t.healthMutex.RUnlock()
	if t.unhealthyCount >= ReaperThreshold {
		log.Warnf("Peer exceeds unhealthy threshold [ %s ]", t)
		return false
	}

	return true
}

func (t *Tracker) String() string {
	return fmt.Sprintf("Tracker{ health: %s, peer: %s }", t.remainingHealth(), t.peer)
}
