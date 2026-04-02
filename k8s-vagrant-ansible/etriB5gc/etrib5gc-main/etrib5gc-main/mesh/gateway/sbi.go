package gateway

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/reogac/utils/httpw"
	"net/http"
)

func (gw *Gateway) onSbi(ctx *gin.Context) {
	var err error
	epId := ctx.GetHeader("epId")
	gwId := ctx.GetHeader("gwId")
	oldUrl := ctx.Request.URL.String()
	log.Debugf("Receive SBI request for endpoint %s[%s]: %s", epId, gwId, oldUrl)
	var url string
	var cli *httpw.Client

	//copy request header
	headers := make(map[string][]string)
	for k, v := range ctx.Request.Header {
		headers[k] = v
	}

	if len(gwId) == 0 || gwId == gw.id { //forward to local endpoint
		var ep *Endpoint
		if ep = gw.epman.findByUuid(epId); ep == nil {
			//TODO: forward to another gw
			log.Warnf("Endpoint %s not found", epId)
			ctx.String(http.StatusNotFound, "Endpoint not found")
			return
		}

		url = fmt.Sprintf("http://%s/%s", ep.sbiUrl, oldUrl[5:])
		cli = ep.cli
		delete(headers, "epId")
		delete(headers, "gwId")
	} else { //forward to another gateway
		var peer *PeerGw
		if peer = gw.getPeer(gwId); peer == nil {
			log.Warnf("GW %s not found", gwId)
			ctx.String(http.StatusBadGateway, "GW not found")
			return

		}
		url = fmt.Sprintf("http://%s/%s", peer.addr, oldUrl[1:])
		cli = peer.cli
		delete(headers, "gwId")
	}

	log.Debugf("Forward request to: %s", url)

	httpReq, _ := http.NewRequest(ctx.Request.Method, url, ctx.Request.Body)
	httpReq.Header = headers

	var httpRsp *http.Response
	if httpRsp, err = cli.SendRequest(httpReq); err != nil {
		log.Warnf("Fail to forward request to endpoint: %+v", err)
		ctx.String(http.StatusServiceUnavailable, "Enpoind not reachable")
		return
	}
	defer httpRsp.Body.Close()

	for k, values := range httpRsp.Header {
		for _, v := range values {
			ctx.Header(k, v)
		}
	}

	ctx.DataFromReader(httpRsp.StatusCode, httpRsp.ContentLength, "application/json", httpRsp.Body, nil)

}

func (gw *Gateway) onAgent(ctx *gin.Context) {
	var err error
	epId := ctx.Param("epId")
	oldUrl := ctx.Request.URL.String()
	log.Debugf("Controller update agent %s [%s]", epId, oldUrl)

	var ep *Endpoint
	if ep = gw.epman.findByUuid(epId); ep == nil {
		log.Warnf("Endpoint %s not found", epId)
		ctx.String(http.StatusNotFound, "Endpoint not found")
		return
	}
	prefixLen := len(epId) + 8
	url := fmt.Sprintf("http://%s/%s", ep.agentUrl, oldUrl[prefixLen:])

	log.Debugf("Forward request to agent: %s", url)

	httpReq, _ := http.NewRequest(ctx.Request.Method, url, ctx.Request.Body)
	httpReq.Header = ctx.Request.Header
	var httpRsp *http.Response
	if httpRsp, err = ep.cli.SendRequest(httpReq); err != nil {
		log.Warnf("Fail to forward request to agent: %+v", err)
		ctx.String(http.StatusServiceUnavailable, "Enpoind not reachable")
		return
	}
	defer httpRsp.Body.Close()

	for k, values := range httpRsp.Header {
		for _, v := range values {
			ctx.Header(k, v)
		}
	}

	ctx.DataFromReader(httpRsp.StatusCode, httpRsp.ContentLength, "application/json", httpRsp.Body, nil)
}
