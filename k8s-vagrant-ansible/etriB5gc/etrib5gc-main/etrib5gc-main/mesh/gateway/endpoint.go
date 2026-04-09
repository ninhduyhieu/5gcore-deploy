package gateway

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/models"
	"fmt"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

const (
	EP_PING_INTERVAL int = 30 //seconds
)

type Endpoint struct {
	*logrus.Entry
	gw                                    *Gateway
	podId                                 string
	id                                    string //generated uuid
	insId                                 string //service specific instance id
	sbiUrl, sbiPrefix, agentUrl, certName string
	labels                                map[string]string
	cli                                   *httpw.Client
	quit                                  chan bool
	regTime                               time.Time
	lastSeen                              time.Time
}

func (gw *Gateway) createEndpoint(req *models.RegistrationRequest) (ep *Endpoint, err error) {
	tmpEp := &Endpoint{
		gw:        gw,
		sbiUrl:    req.SbiUrl,
		sbiPrefix: req.SbiPrefix,
		agentUrl:  req.AgentUrl,
		certName:  req.CertName,
		regTime:   time.Now(),
		lastSeen:  time.Now(),
		labels:    make(map[string]string),
	}

	if len(req.PodName) > 0 { //endpoint send a PodName
		if err = gw.updateEpInfo(req.PodName, req.PodNameSpace, tmpEp); err != nil { //get endpoint pod's metadata (labels)
			err = utils.WrapError("Get pod's metadata", err)
			return
		}
	} else {
		if len(tmpEp.sbiUrl) == 0 {
			err = fmt.Errorf("Empty Sbi URL")
			return
		}
		if len(tmpEp.agentUrl) == 0 {
			err = fmt.Errorf("Empty Agent URL")
			return
		}

		for k, v := range req.Labels {
			tmpEp.labels[k] = v
		}
	}

	tmpEp.Entry = logctx.Entry(logrus.Fields{
		"endpoint": tmpEp.sbiUrl,
	})
	ep = tmpEp
	ep.cli = httpw.NewClient(gw.cert, gw.caPool, req.CertName)
	return
}

func (ep *Endpoint) loop(wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Duration(EP_PING_INTERVAL) * time.Second)
	notify := false
LOOP:
	for {
		select {
		case <-ticker.C:
			if !ep.ping() {
				notify = true
				break LOOP
			}
			ep.lastSeen = time.Now()
		case <-ep.quit:
			break LOOP
		}
	}

	ep.gw.removeEp(ep, notify)
	ep.Infof("Endpoint removed")
}
func (ep *Endpoint) close() {
	ep.Tracef("Close")
	close(ep.quit)
}

func (ep *Endpoint) ping() bool {
	url := fmt.Sprintf("%s/ping", ep.agentUrl)
	if rsp, _, err := ep.cli.Send(http.MethodGet, url, nil); err != nil || rsp.StatusCode != http.StatusOK {
		ep.Errorf("Ping %s failed: %+v", url, err)
		return false
	}
	ep.Tracef("Endpoint is pinged")
	return true
}

func (ep *Endpoint) toModel() models.Endpoint {
	return models.Endpoint{
		Id:        ep.id,
		Labels:    ep.labels,
		InsId:     ep.insId,
		GwId:      ep.gw.id,
		SbiUrl:    ep.sbiUrl,
		SbiPrefix: ep.sbiPrefix,
		Name:      ep.certName,
	}
}
