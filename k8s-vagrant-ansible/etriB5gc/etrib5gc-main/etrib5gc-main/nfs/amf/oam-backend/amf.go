package backend

import (
	"context"
	"encoding/json"
	"etrib5gc/internal/fsm"
	"etrib5gc/nfs/amf/types"
	"etrib5gc/nfs/amf/uecontext"
	"fmt"
	"github.com/reogac/utils/oam"
	"github.com/urfave/cli/v3"
)

type OamAmfHandler struct {
	nextContext *oam.HandlerContext //next context
}

var amfCmds map[string]cli.Command = map[string]cli.Command{
	"fsm-stats": {
		Name:        "fsm-stats",
		Usage:       "Show the 5GMM FSM statistics",
		Description: "Show the statistics of mobility management state machine of UEs",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			stats := h.getFsmStats()
			info, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Fprintf(cmd.Writer, string(info))
			return nil
		},
	},
	"fsm-worker-stats": {
		Name:        "fsm-worker-stats",
		Usage:       "Show the statistics of worker pool for 5GMM FSM",
		Description: "Show the statistics of the worker pool for executing handlers of the mobility management state machine of UEs",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			stats := h.getFsmWorkerStats()
			info, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Fprintf(cmd.Writer, string(info))
			return nil
		},
	},
	"pub-worker-stats": {
		Name:        "pub-worker-stats",
		Usage:       "Show the statistics of the public worker pool",
		Description: "Show the statistics of the worker pool for executing parallel tasks",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			stats := h.getPublicWorkerStats()
			info, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Fprintf(cmd.Writer, string(info))
			return nil
		},
	},

	"count-ue": {
		Name:        "count-ue",
		Usage:       "Show the number of UEs",
		Description: "Show the number of UEs",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			fmt.Fprintf(cmd.Writer, "Number of UEs: %d\n", h.getNumUeContexts())
			return nil
		},
	},

	"list-ue": {
		Name:        "list-ue",
		Usage:       "List Ue contexts",
		Description: "List all current UeContexts",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			ues := h.listUeContexts()
			if len(ues) == 0 {
				return fmt.Errorf("No UE contexts found")
			} else {
				info, _ := json.MarshalIndent(ues, "", "  ")
				fmt.Fprintf(cmd.Writer, "%s\n", string(info))
			}
			return nil
		},
	},
	"view-ue": {
		Name:        "view-ue",
		Description: "View UeContext information",
		Usage:       "View Ue context information",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "supi",
				Usage:    "SUPI string",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			supi := cmd.String("supi")
			if ueInfo := h.viewUeContext(supi); ueInfo == nil {
				return fmt.Errorf("UE context for supi=%s not found", supi)
			} else {
				info, _ := json.MarshalIndent(ueInfo, "", "  ")
				fmt.Fprintf(cmd.Writer, string(info))
			}
			return nil
		},
	},

	"deregister-ue": {
		Name:        "deregister-ue",
		Usage:       "Deregister Ue",
		Description: "Deregister a UE from the network with the specified SUPI",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "cause",
				Usage: "Integer 0,1,2",
				Value: 0,
			},
			&cli.StringFlag{
				Name:     "supi",
				Usage:    "SUPI string",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			cause := uint8(cmd.Int("cause"))
			supi := cmd.String("supi")

			success := h.deregisterUe(supi, cause)
			if success {
				fmt.Fprintf(cmd.Writer, "Start deregistration for UE %s", supi)
			} else {
				return fmt.Errorf("Failed to trigger deregistration for UE %s", supi)
			}
			return nil
		},
	},
	"select-ue": {
		Name:        "select-ue",
		Usage:       "Switch to Ue context",
		Description: "Switch to Ue context",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "tmsi",
				Usage:    "select-ue --tmsi <tmsi>",
				Required: true,
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			h := ctx.Value("handler").(*OamAmfHandler)
			ueId := cmd.Int("tmsi")
			if srvCtx, err := h.getUeContext(uint32(ueId)); err != nil {
				return err
			} else {
				h.nextContext = srvCtx
			}
			return nil
		},
	},
}

func (b *OamAmfHandler) NextContext() *oam.HandlerContext {
	return b.nextContext
}

//build context for Ue
func (b *OamAmfHandler) getUeContext(tmsi uint32) (ctx *oam.HandlerContext, err error) {
	//Find UE first
	ueCtx := uecontext.FindUeByTmsi(tmsi)
	if ueCtx == nil {
		err = fmt.Errorf("UE with TMSI=%d  not found", tmsi)
		return
	}
	ctxId := fmt.Sprintf("ue:%d", tmsi)
	ctx = oam.NewHandlerContext(ctxId, &OamUeHandler{
		ueCtx: ueCtx,
	}, ueCmds, nil)
	return
}

// ListUeContexts returns a list of UE contexts
func (b *OamAmfHandler) getNumUeContexts() int {
	return uecontext.OamGetNumUes()
}

// ListUeContexts returns a list of UE contexts
func (b *OamAmfHandler) listUeContexts() []types.UeContextShortInfo {
	return uecontext.OamGetUeList()
}

// ViewUeContext  view details of a single UeContext
func (b *OamAmfHandler) viewUeContext(supi string) *types.UeContextInfo {
	return uecontext.OamGetUeInfo(supi)
}

// DeregisterUe deregisters a UE with the given SUPI
func (b *OamAmfHandler) deregisterUe(supi string, cause uint8) bool {
	return uecontext.OamDeregisterUe(supi, cause)
}

// Get stats of 5GMM state machine
func (b *OamAmfHandler) getFsmStats() *fsm.FsmInfo {
	return uecontext.OamGetFsmStats()
}

// Get stats of 5GMM state machine worker pool
func (b *OamAmfHandler) getFsmWorkerStats() types.WorkerInfo {
	return uecontext.OamGetFsmWorkerStats()
}

// Get stats of public  worker pool
func (b *OamAmfHandler) getPublicWorkerStats() types.WorkerInfo {
	return uecontext.OamGetPublicWorkerStats()
}
