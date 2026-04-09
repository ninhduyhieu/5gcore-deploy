package sepp

import (
	"etrib5gc/mesh"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

//handle request comming to SEPP from another SEPP
func (nf *SEPP) handleIngress(c *gin.Context) {
	nf.Infof("Handle Ingress")
	if nfId := c.GetHeader(TARGET_ID_HEADER); len(nfId) > 0 { //remote SEPP send nfId
		if ep := nf.localEndpoints.getEndpoint(nfId); ep == nil {
			c.JSON(http.StatusNotFound, "Endpoint not found")
			return
		} else {
			nf.Infof("Found LocalEndpoint with nfId=%s", nfId)
			nf.forwardToLocalEndpoint(ep, false, c)
		}
	} else { //need to pick the local NF with service Id
		serviceId := c.GetHeader(SERVICE_ID_HEADER)
		if len(serviceId) == 0 {
			c.JSON(http.StatusBadRequest, "No service id")
			return
		}
		// Fomr ServiceId -> pick an NF instance for the service
		if ep := nf.pickLocalEndpoint(serviceId); ep == nil {
			c.JSON(http.StatusNotFound, "Endpoint not found for "+serviceId)
			return
		} else {
			nf.Infof("Pick a LocalEndpoint for service %s with nfId=%s", serviceId, ep.nfId)
			nf.forwardToLocalEndpoint(ep, true, c)
		}
	}
	c.JSON(http.StatusInternalServerError, "not implement")
}

func (nf *SEPP) forwardToLocalEndpoint(ep *LocalEndpoint, isNew bool, c *gin.Context) {
	// rewrite callback [may add a new RemoteEndpoint]
	if h := c.GetHeader(CALLBACK_HEADER); len(h) > 0 {
		nf.Infof("There is a callback: %s", h)
		if callback, err := nf.rewriteIngressCallback(h); err != nil {
			nf.Errorf("Fail to rewrite callback: %+v", err)
			c.String(http.StatusBadRequest, "Bad callback URI")
			return
		} else {
			c.Request.Header.Set(CALLBACK_HEADER, callback)
		}
	}
	builder := func(orgReq *http.Request) func(string) (*http.Request, error) {
		return func(addr string) (*http.Request, error) {
			oldUrl := orgReq.URL.String()
			url := fmt.Sprintf("http://%s/%s", addr, oldUrl[1:]) //strip the first charater from the original path
			if req, err := http.NewRequest(orgReq.Method, url, orgReq.Body); err != nil {
				return nil, err
			} else {
				req.Header = orgReq.Header
				return req, nil
			}
		}
	}(c.Request)
	rsp, err := ep.cli.SendHttpRequest(builder)
	if err != nil {
		nf.Errorf("Fail to forward request: %+v", err)
		c.JSON(http.StatusServiceUnavailable, "Endpoint is unreachable")
		return
	}
	for k, values := range rsp.Header {
		for _, v := range values {
			c.Header(k, v)
		}
	}

	if isNew {
		nf.Infof("Set nfId=%s for response header", ep.nfId)
		c.Header(TARGET_ID_HEADER, ep.nfId)
	}
	//write response
	c.DataFromReader(rsp.StatusCode, rsp.ContentLength, "application/json", rsp.Body, nil)
}

func (nf *SEPP) pickLocalEndpoint(serviceId string) *LocalEndpoint {
	if cli, err := mesh.Consumer(serviceId, nil); err != nil {
		nf.Errorf("Fail to pick an endpoint: %+v", err)
	} else {
		if ep := nf.localEndpoints.findInstance(cli.GetId()); ep == nil {
			ep = &LocalEndpoint{
				cli: cli,
			}
			nf.Infof("Create a new local endpoint")
			nf.localEndpoints.add(ep)
			return ep
		} else {
			nf.Infof("Found old local endpoint")
			return ep
		}
	}
	return nil
}
