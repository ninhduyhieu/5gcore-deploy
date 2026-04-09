package main

import (
	"etrib5gc/mesh/gateway"
	"etrib5gc/nfs/app"
	"fmt"
	"github.com/urfave/cli/v3"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ctrl",
		Usage:   "Controller name for certificate verification",
		Sources: cli.EnvVars("CONTROLLER_NAME"),
	},
}

func main() {
	nf := new(gateway.Gateway)
	usage := "6G Core Network Gateway"
	if err := app.RunApp("GW", usage, flags, nf); err != nil {
		fmt.Printf("Failed to run GW: %+v\n", err)
	}
}
