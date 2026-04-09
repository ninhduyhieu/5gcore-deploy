package backend

import (
	"context"
	"encoding/json"
	"etrib5gc/nfs/amf/types"
	"etrib5gc/nfs/amf/uecontext"
	"fmt"
	"github.com/urfave/cli/v3"
)

var ueCmds map[string]cli.Command = map[string]cli.Command{
	"view": {
		Name:        "view",
		Usage:       "View Ue context",
		Description: "View UeContext information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			//get backend handler
			handler := ctx.Value("handler").(*OamUeHandler)
			//get UeContext information
			info := handler.viewUeContext()
			// marshal to bytes
			infoBytes, _ := json.MarshalIndent(&info, "", "  ")
			//write as string
			fmt.Fprintf(cmd.Writer, string(infoBytes))
			return nil
		},
	},
	"deregister": {
		Name:        "deregister",
		Usage:       "Deregister Ue",
		Description: "Trigger deregistration",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "cause",
				Usage:    "Integer 0,1,2",
				Required: true,
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			//get backend handler
			handler := ctx.Value("handler").(*OamUeHandler)
			cause := cmd.Int("cause")
			//get UeContext information
			if success := handler.deregisterUe(uint8(cause)); success {
				fmt.Fprintf(cmd.Writer, "Degistration triggered")
			}
			return nil
		},
	},
	"release": {
		Name:        "release",
		Usage:       "Release UeContext at RAN (CM-IDLE)",
		Description: "Trigger to command RAN to release UeContext",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "nonGpp",
				Usage: "for Non-Gpp access",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			//get backend handler
			handler := ctx.Value("handler").(*OamUeHandler)
			//TODO: get isGpp flag
			handler.releaseUe(true)
			//write as string
			fmt.Fprintf(cmd.Writer, "Release UeContext is triggered")
			return nil
		},
	},
	"ps-release-all": {
		Name:        "ps-release-all",
		Usage:       "Release all Pdu sessions",
		Description: "Trigger release of all Pdu sessions",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			//get backend handler
			handler := ctx.Value("handler").(*OamUeHandler)
			handler.releaseAllSessions()
			//write as string
			fmt.Fprintf(cmd.Writer, "Release all Pdu session event triggered")
			return nil
		},
	},
	"ps-release": {
		Name:        "ps-release",
		Usage:       "Release a Pdu session",
		Description: "Trigger Pdu session release",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "session",
				Usage:    "Session Id (1-16)",
				Required: true,
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			//get backend handler
			handler := ctx.Value("handler").(*OamUeHandler)
			sessionId := cmd.Int("session")
			//get UeContext information
			handler.releaseSession(sessionId)
			fmt.Fprintf(cmd.Writer, "Release session event triggered")
			return nil
		},
	},
}

type OamUeHandler struct {
	ueCtx *uecontext.UeContext
}

func (b *OamUeHandler) viewUeContext() types.UeContextInfo {
	return b.ueCtx.WriteInfo()
}

func (b *OamUeHandler) deregisterUe(cause uint8) bool {
	return uecontext.OamDeregister(b.ueCtx, cause)
}
func (b *OamUeHandler) releaseUe(isGpp bool) {
	uecontext.OamRelease(b.ueCtx, isGpp)
}
func (b *OamUeHandler) releaseAllSessions() {
	//TODO:release all sessions
}
func (b *OamUeHandler) releaseSession(sId int) {
	//TODO: release a session
}
