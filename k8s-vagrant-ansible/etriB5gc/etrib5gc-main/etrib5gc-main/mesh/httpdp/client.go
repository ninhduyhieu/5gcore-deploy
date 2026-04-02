package httpdp

import (
	"etrib5gc/mesh/registry"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/utils"
	"net/http"
)

var STICKINESS_HEADER string = http.CanonicalHeaderKey("xxx-stickiness")
var HOST_HEADER string = http.CanonicalHeaderKey("serviceId")

type StatelessService struct {
	lb func() *registry.Endpoint
}

type Client struct {
	endpoint         *registry.Endpoint
	serviceId        string
	statelessService *StatelessService
	stickiness       string
}

func NewStatelessClient(serviceId string, group *registry.MatchedGroup) (sbiCli *Client) {
	sbiCli = &Client{
		serviceId: serviceId,
		statelessService: &StatelessService{
			lb: group.Select,
		},
	}
	return
}

func NewStatefulClient(serviceId string, ep *registry.Endpoint) (sbiCli *Client) {
	sbiCli = &Client{
		endpoint:  ep,
		serviceId: serviceId,
	}
	return
}

func NewClientWithEndpoint(ep *registry.Endpoint, stickiness string) (sbiCli *Client) {
	sbiCli = &Client{
		endpoint:   ep,
		stickiness: stickiness,
	}
	return
}

func (c *Client) GetId() string {
	if c.endpoint != nil {
		return c.endpoint.Id()
	}
	return c.serviceId
}

func (c *Client) getClient() (*registry.EpClient, error) {
	if c.endpoint != nil {
		return c.endpoint.Client(), nil
	} else {
		if ep := c.statelessService.lb(); ep == nil {
			return nil, fmt.Errorf("Can't pick and endpoint")
		} else {
			return ep.Client(), nil
		}

	}

}

func (c *Client) SendHttpRequest(builder func(addr string) (*http.Request, error)) (rsp *http.Response, err error) {
	var req *http.Request
	var cli *registry.EpClient
	//get client
	if cli, err = c.getClient(); err != nil {
		err = fmt.Errorf("Create http client failed: %+v", err)
		return
	}

	if req, err = builder(cli.Addr); err != nil {
		err = utils.WrapError("Build http request", err)
		return
	}
	//set routing header
	if r := cli.Route; r != nil {
		req.Header.Add("gwId", r.GwId)
		req.Header.Add("epId", r.EpId)
	}

	//set session stickiness header
	if len(c.stickiness) > 0 {
		req.Header.Add(STICKINESS_HEADER, c.stickiness)
	}

	//set host to be service name
	if len(c.serviceId) > 0 {
		req.Header.Add(HOST_HEADER, c.serviceId)
	}

	//send request and receive a response
	if rsp, err = cli.Cli.SendRequest(req); err != nil {
		//err = utils.WrapError("Send http request", err)
		return
	} else {
		//check for stickiness header
		if h, ok := rsp.Header[STICKINESS_HEADER]; ok {
			c.stickiness = h[0]
		}
	}
	return

}

// TODO: add re-transmission after a failure
func (c *Client) Send(sbiRequest *sbi.Request) (rsp *sbi.Response, err error) {
	//send the message here
	var httpResponse *http.Response
	var httpRequest *http.Request
	var cli *registry.EpClient

	//get client
	if cli, err = c.getClient(); err != nil {
		err = fmt.Errorf("Create http client failed: %+v", err)
		return
	}
	if httpRequest, err = sbi.BuildHttpRequest(sbiRequest, cli.Addr); err != nil {
		return
	}
	//set routing header
	if r := cli.Route; r != nil {
		httpRequest.Header.Add("gwId", r.GwId)
		httpRequest.Header.Add("epId", r.EpId)
	}

	//set session stickiness header
	if len(c.stickiness) > 0 {
		httpRequest.Header.Add(STICKINESS_HEADER, c.stickiness)
	}

	//set host to be service name
	if len(c.serviceId) > 0 {
		httpRequest.Header.Add(HOST_HEADER, c.serviceId)
	}

	//send request and receive a response
	if httpResponse, err = cli.Cli.SendRequest(httpRequest); err != nil {
		//err = utils.WrapError("Send http request", err)
		return
	} else {
		//check for stickiness header
		if h, ok := httpResponse.Header[STICKINESS_HEADER]; ok {
			c.stickiness = h[0]
		}
		rsp = sbi.CreateResponse(httpResponse)
	}
	return
}
