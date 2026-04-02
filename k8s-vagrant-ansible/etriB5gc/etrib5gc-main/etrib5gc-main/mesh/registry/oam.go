package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/reogac/utils/oam"
	"github.com/urfave/cli/v3"
)

const (
	AGENT_CTX string = "agent"
)

var cmds map[string]cli.Command = map[string]cli.Command{
	"agentInfo": {
		Name:  "agentInfo",
		Usage: "Show agent information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*Registry)
			dat := h.getAgentInfo()
			info, _ := json.MarshalIndent(dat, "", "  ")
			fmt.Fprintf(cmd.Writer, string(info))
			return nil
		},
	},
}

func (reg *Registry) CreateOamHandler() *oam.OamHandler {
	return oam.NewOamHandler("agent", AGENT_CTX, func(ctxId string) *oam.HandlerContext {
		if ctxId == AGENT_CTX {
			return oam.NewHandlerContext("agent", reg, cmds, nil)
		}
		return nil
	})
}

func (reg *Registry) CreateOamHandlerContext() *oam.HandlerContext {
	return oam.NewHandlerContext(AGENT_CTX, reg, cmds, nil)
}

type AgentInfo struct {
	Id string
}

func (reg *Registry) getAgentInfo() *AgentInfo {
	return &AgentInfo{
		Id: reg.id,
	}
}
