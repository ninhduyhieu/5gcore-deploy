package oambackend

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v3"
)

var cmds map[string]cli.Command = map[string]cli.Command{
	"list-ep": {
		Name:        "list-ep",
		Usage:       "List endpoints",
		Description: "List all registered endpoints",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*GwHandler)
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

type GwHandler struct {
	api OamApi
}

func (b *GwHandler) listEndpoints() (infos []EndpointInfo) {
	return b.api.GetEndpointInfos()
}
