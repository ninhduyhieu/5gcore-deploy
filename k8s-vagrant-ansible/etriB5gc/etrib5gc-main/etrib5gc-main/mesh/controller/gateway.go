package controller

import (
	"etrib5gc/mesh/models"
	"fmt"
	"github.com/reogac/utils/httpw"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	GW_PING_INTERVAL int = 60 //seconds
)

type Gateway struct {
	url       string
	labels    map[string]string
	name      string
	id        string
	quit      chan bool
	gwman     *GatewayManager
	regTime   time.Time
	lastSeen  time.Time
	endpoints map[string]*Endpoint
	cli       *httpw.Client
	mutex     sync.RWMutex
}

func (ctrl *Controller) newGateway(req *models.GwRegistrationRequest) *Gateway {
	id := uuid.New().String()
	gw := &Gateway{
		gwman:     ctrl.gwman,
		id:        id,
		labels:    req.Labels,
		regTime:   time.Now(),
		lastSeen:  time.Now(),
		url:       req.Url,
		name:      req.Name,
		cli:       httpw.NewClient(ctrl.cert, ctrl.caPool, req.Name),
		quit:      make(chan bool),
		endpoints: make(map[string]*Endpoint),
	}
	return gw
}

func (gw *Gateway) addEp(ep *Endpoint) {
	gw.mutex.Lock()
	defer gw.mutex.Unlock()
	gw.endpoints[ep.id] = ep
}

func (gw *Gateway) removeEp(ep *Endpoint) {
	gw.mutex.Lock()
	defer gw.mutex.Unlock()
	delete(gw.endpoints, ep.id)
}

func (gw *Gateway) loop(wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Duration(GW_PING_INTERVAL) * time.Second)
LOOP:
	for {
		select {
		case <-ticker.C:
			if !gw.ping() {
				break LOOP
			}
			gw.lastSeen = time.Now()
		case <-gw.quit:
			break LOOP
		}
	}

	gw.gwman.removeGw(gw)
}

func (gw *Gateway) close() {
	close(gw.quit)
}

func (gw *Gateway) ping() bool {
	url := fmt.Sprintf("%s/ping", gw.url)
	if rsp, _, err := gw.cli.Send(http.MethodGet, url, nil); err != nil || rsp.StatusCode != http.StatusOK {
		if err != nil {
			log.Errorf("Fail to ping gateway %s: %+v", gw.url, err)
		} else {
			log.Errorf("Fail to ping gateway %s", gw.url)
		}
		return false
	}
	return true
}

func (gw *Gateway) getEndpoints() (endpoints []*Endpoint) {
	gw.mutex.RLock()
	defer gw.mutex.RUnlock()
	if l := len(gw.endpoints); l > 0 {
		endpoints = make([]*Endpoint, l)
		i := 0
		for _, ep := range gw.endpoints {
			endpoints[i] = ep
			i++
		}
	}
	return
}
