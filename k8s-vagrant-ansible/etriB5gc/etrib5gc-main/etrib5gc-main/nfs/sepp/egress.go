package sepp

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// handle requests comming to SEPP toward another SEPP
func (nf *SEPP) handleEgress(c *gin.Context) {
	nf.Infof("Handle Egress")
	if localId := c.GetHeader(TARGET_ID_HEADER); len(localId) > 0 {
		ep := nf.remoteEndpoints.getEndpoint(localId)
		if ep == nil {
			c.String(http.StatusBadGateway, "Can't find destination")
			return
		}
		nf.Infof("Found remote endpoint with localId=%s", localId)
		nf.forwardToRemoteEndpoint(ep, c)

	} else {
		serviceId := c.GetHeader(SERVICE_ID_HEADER)
		if len(serviceId) == 0 {
			nf.Errorf("Receive a request without service Id")
			c.String(http.StatusBadRequest, "No target service")
			return
		}
		//From ServiceId -> PlmnId -> pick a SEPP peer
		peer, err := nf.pickPeer(serviceId)
		if err != nil {
			nf.Errorf("Fail to pick a peer for %s:%+v", serviceId, err)
			c.String(http.StatusServiceUnavailable, "No endpoint found")
			return
		}

		nf.Infof("Found peer %s, create a remote endpoint then forward request", peer.url)
		nf.forwardToRemoteEndpoint(&RemoteEndpoint{
			sepp: peer,
		}, c) //new RemoteEndpoint without localId and nfId
	}
}

func (nf *SEPP) forwardToRemoteEndpoint(ep *RemoteEndpoint, c *gin.Context) {

	//1. build a request
	oldUrl := c.Request.URL.String()
	//a. copy request header
	headers := make(map[string][]string)
	for k, v := range c.Request.Header {
		if k != TARGET_ID_HEADER {
			headers[k] = v
		}
	}
	//b. make request
	url := fmt.Sprintf("http://%s/%s", ep.sepp.url, oldUrl[SEPP_SBI_PREFIX_OFFSET:])

	httpReq, _ := http.NewRequest(c.Request.Method, url, c.Request.Body)
	httpReq.Header = headers

	//c. write target
	if len(ep.nfId) > 0 {
		nf.Infof("Add nfId to request header")
		httpReq.Header.Add(TARGET_ID_HEADER, ep.nfId)
	}
	//d. rewrite callback [may add a new LocalEndpoint]
	if h := httpReq.Header.Get(CALLBACK_HEADER); len(h) > 0 {
		if callback, err := nf.rewriteEgressCallback(h); err != nil {
			nf.Errorf("Fail to rewrite callback: %+v", err)
			c.String(http.StatusBadRequest, "Bad callback URI")
			return
		} else {
			httpReq.Header.Set(CALLBACK_HEADER, callback)
		}
	}

	//2. send request
	var httpRsp *http.Response
	var err error
	if httpRsp, err = ep.sepp.cli.SendRequest(httpReq); err != nil {
		nf.Warnf("Fail to forward request to remote SEPP: %+v", err)
		c.String(http.StatusServiceUnavailable, "Remote SEPP not reachable")
		return
	}
	defer httpRsp.Body.Close()

	if len(ep.localId) == 0 { //new discoveried NF
		if h := httpRsp.Header.Get(TARGET_ID_HEADER); len(h) == 0 {
			//remote SEPP does not send target Id
			nf.Errorf("H-SEPP did not send target nfId", err)
			c.String(http.StatusServiceUnavailable, "Enpoind not reachable")
			return
		} else {
			nf.Infof("Set nfId for remote endpoint: %s", h)
			ep.nfId = h
		}
		if old := nf.remoteEndpoints.findByRemoteId(ep.getRemoteId()); old != nil {
			nf.Infof("Remote endpoint existed")
			//remote endpoint existed
			ep = old
		} else {
			//add new remote endpoint
			nf.remoteEndpoints.add(ep)
			nf.Infof("Add new remote endpoint nfId=%s", ep.nfId)
		}
		nf.Infof("Set response header with localId=%s", ep.localId)
		//NOTE: now the remote endpoint should have a local Id
		//attach localId to the response
		c.Header(TARGET_ID_HEADER, ep.localId)

	}
	for k, values := range httpRsp.Header {
		if k != TARGET_ID_HEADER {
			for _, v := range values {
				c.Header(k, v)
			}
		}
	}

	//3. write response
	c.DataFromReader(httpRsp.StatusCode, httpRsp.ContentLength, "application/json", httpRsp.Body, nil)
}

func (nf *SEPP) pickPeer(serviceId string) (*SeppPeer, error) {
	if plmnId := parsePlmnId(serviceId); plmnId == nil {
		return nil, fmt.Errorf("Fail to parse PlmnId from service Id")
	} else {
		if p := nf.peers.pickPeer(*plmnId); p == nil {
			return nil, fmt.Errorf("Can't find a SEPP for %s", serviceId)
		} else {
			return p, nil
		}
	}
}
