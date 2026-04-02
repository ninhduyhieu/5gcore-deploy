package controller

import (
	"bytes"
	"encoding/json"
	"etrib5gc/mesh/models"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const (
	EP_PING_INTERVAL int = 10 //seconds
)

type Endpoint struct {
	*logrus.Entry
	id        string //generated uuid
	insId     string //service specific instance id
	sbiActive bool
	labels    map[string]string
	quit      chan bool
	services  map[string]*Service //list of subscribed services
	servings  []*Service          //list of services the endpoint is serving
	regTime   time.Time
	config    []byte //configuration that can be shared to other endpoints
	gw        *Gateway
}

func (ctrl *Controller) createEndpoint(gw *Gateway) (ep *Endpoint) {
	//generate id
	id := uuid.New().String()
	ep = &Endpoint{
		gw:       gw,
		labels:   make(map[string]string),
		services: make(map[string]*Service),
		quit:     make(chan bool),
		id:       id,
		regTime:  time.Now(),
	}
	for k, v := range gw.labels {
		ep.labels[k] = v
	}
	return
}

func (ep *Endpoint) toModel() models.Endpoint {
	return models.Endpoint{
		Id:     ep.id,
		Labels: ep.labels,
		InsId:  ep.insId,
		Config: ep.config,
		GwId:   ep.gw.id,
	}
}

// check if an endpoint is serving a service?
func (ep *Endpoint) isServing(s *Service) bool {
	//check for labels matching to selectors
	for k, v := range s.selectors {
		if v1, ok := ep.labels[k]; !ok {
			return false
		} else if strings.Compare(v1, v) != 0 {
			return false
		}
	}

	return true
}

func (ep *Endpoint) notifyEpLeft(lefter *Endpoint) {
	if ep == lefter {
		return
	}
	ep.sendUpdate(&models.EndpointUpdates{
		Left: []string{lefter.id},
	})
}

func (ep *Endpoint) notifyEpJoin(joiner *Endpoint) {
	if ep == joiner {
		return
	}
	ep.sendUpdate(&models.EndpointUpdates{
		Join: []models.Endpoint{
			joiner.toModel(),
		},
	})
}

func (ep *Endpoint) sendUpdate(msg *models.EndpointUpdates) {
	buf, _ := json.Marshal(msg)
	body := bytes.NewBuffer(buf)
	url := fmt.Sprintf("%s/update/%s", ep.gw.url, ep.id)

	cli := ep.gw.cli
	if rsp, rspBody, err := cli.Send(http.MethodPost, url, body); err != nil {
		ep.Errorf("Send update notification failed:%+v", err)
	} else if rsp.StatusCode != http.StatusCreated {
		ep.Warnf("Update not processed by endpoint: %s:%d:%s: %s", url, rsp.StatusCode, rsp.Status, string(rspBody))
	}
}

// remove the endpoint from subsciber lists from services
func (ep *Endpoint) unsubscribeServices() {
	for _, service := range ep.services {
		service.removeSub(ep)
	}
}

// remove the endpoint from services it serves
func (ep *Endpoint) detachServices() {
	for _, service := range ep.servings {
		service.removeEndpoint(ep)
	}
}
