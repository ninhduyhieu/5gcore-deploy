package main

import (
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/pingapp/pingservice"
	"github.com/reogac/sbi/models"
	"time"
)

type Producer struct {
	service *PingService
}

func (p *Producer) HandlePing(body *models.PingRequest) (rsp *models.PingResponse, prob *models.ProblemDetails) {
	fmt.Printf("Receive ping message: %s\n", body.Message)
	rsp = &models.PingResponse{
		Message: fmt.Sprintf("ECHO: %s", body.Message),
		Nonce:   body.Nonce,
		Time:    time.Now().Format(time.UnixDate),
		From:    p.service.appName,
	}
	return
}

func (p *Producer) HandleForward(body *models.PingFwRequest) (rsp *models.PingResponse, prob *models.ProblemDetails) {
	fmt.Printf("Receive ping forward message[to %s]: %s\n", body.Message, body.Service)
	var cli sbi.ConsumerClient
	var err error
	//1. discover the service
	if cli, err = mesh.Consumer(body.Service, nil); err != nil {
		prob = &models.ProblemDetails{
			Status: 500,
			Detail: fmt.Sprintf("Fail to discover service %s: %+v", body.Service, err),
		}
		return
	}
	//2. ping the service
	if rsp, err = pingservice.Ping(cli, &models.PingRequest{
		Message: body.Message,
		Time:    body.Time,
		Nonce:   body.Nonce,
	}); err != nil {
		prob = &models.ProblemDetails{
			Status: 500,
			Detail: fmt.Sprintf("Fail to ping service %s: %+v", body.Service, err),
		}
	}
	//3 return the response

	return
}
