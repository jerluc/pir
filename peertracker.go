package pir

import (
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

// TODO: Use some kind of exponential backoff and retention

const (
	HealthCheckInterval = 2 * time.Second
	HealthCheckTimeout  = 2 * time.Second
	ReaperThreshold     = 3
)

type Tracker struct {
	peer           *Peer
	healthCheck    func() bool
	deathChan      chan bool
	unhealthyCount int
	healthMutex    sync.RWMutex
	reaper         func(*Peer)
}

func NewTracker(peer *Peer, healthCheck func() bool, reaper func(*Peer)) *Tracker {
	tracker := &Tracker{peer, healthCheck, make(chan bool, 1),
		0, sync.RWMutex{}, reaper}
	go tracker.track()
	return tracker
}

func (t *Tracker) MarkHealthy() {
	t.healthMutex.Lock()
	defer t.healthMutex.Unlock()

	t.unhealthyCount -= 1
	if t.unhealthyCount < 0 {
		t.unhealthyCount = 0
	}
}

func (t *Tracker) MarkUnhealthy() {
	t.healthMutex.Lock()
	defer t.healthMutex.Unlock()

	t.unhealthyCount += 1
}

func (t *Tracker) Kill() {
	t.deathChan <- true
}

func (t *Tracker) track() {
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
