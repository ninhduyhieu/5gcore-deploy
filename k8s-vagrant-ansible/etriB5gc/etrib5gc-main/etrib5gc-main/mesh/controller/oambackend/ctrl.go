package oambackend

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v3"
)

var cmds map[string]cli.Command = map[string]cli.Command{
	"list-gw": {
		Name:        "list-gw",
		Usage:       "List gateways",
		Description: "List all registered gateways",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*CtrlHandler)
			gateways := h.listGateways()
			if len(gateways) == 0 {
				return fmt.Errorf("No registered gateways")
			} else {
				info, _ := json.MarshalIndent(gateways, "", "  ")
				fmt.Fprintf(cmd.Writer, string(info))
			}
			return nil
		},
	},
	"list-svc": {
		Name:        "list-svc",
		Description: "List all services",
		Usage:       "List all defined services",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*CtrlHandler)
			services := h.listServices()
			if len(services) == 0 {
				return fmt.Errorf("No service defined")
			} else {
				info, _ := json.MarshalIndent(services, "", "  ")
				fmt.Fprintf(cmd.Writer, string(info))
			}
			return nil
		},
	},
	"list-ep": {
		Name:        "list-ep",
		Usage:       "List endpoints",
		Description: "List all registered endpoints",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*CtrlHandler)
			endpoints := h.listEndpoints()
			if len(endpoints) == 0 {
				return fmt.Errorf("No registered endpoints")
			} else {
				info, _ := json.MarshalIndent(endpoints, "", "  ")
				fmt.Fprintf(cmd.Writer, string(info))
			}
			return nil
		},
	},
}

type CtrlHandler struct {
	api OamApi
}

func (b *CtrlHandler) listEndpoints() (infos []EndpointInfo) {
	return b.api.GetEndpointInfos()
}

func (b *CtrlHandler) listServices() (infos []ServiceInfo) {
	return b.api.GetServiceInfos()
}

func (b *CtrlHandler) listGateways() (infos []GatewayInfo) {
	return b.api.GetGatewayInfos()
}
