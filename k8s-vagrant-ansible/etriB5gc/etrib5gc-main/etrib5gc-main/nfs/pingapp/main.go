package main

import (
	"encoding/json"
	"etrib5gc/mesh"
	"etrib5gc/nfs/app"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/pingapp/pingservice"
	"github.com/reogac/utils"
	"github.com/urfave/cli"
)

var flags = []cli.Flag{
	cli.StringFlag{
		Name:  "config, c",
		Usage: "Load configuration from `FILE`",
	},
}

func main() {
	nf := new(PingService)
	usage := "Ping Server"
	if err := app.RunNfApp("pingapp", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run pingapp: %+v\n", err)
	}
}

type PingService struct {
	appName  string
	producer *Producer
}

func (s *PingService) Init(ctx *app.Context) error {
	var cfg Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	s.appName = cfg.Name

	//create producer
	s.producer = &Producer{
		service: s,
	}

	ctx.SetMeshOptions(&mesh.Options{
		Config: &cfg.AgentConfig,
	})
	return nil
}

func (s *PingService) Run() error {
	//no nothing
	return nil
}

func (s *PingService) Stop() {
	//do nothing
}

func (s *PingService) GetSbiServices() []sbi.SbiService {
	return []sbi.SbiService{pingservice.Service(s.producer)}
}
