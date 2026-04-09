package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/smf/service"
	"fmt"
	"github.com/urfave/cli/v3"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:  "exp, e",
		Usage: "`FILE` to record experiment results",
	},
}

func main() {
	nf := new(service.SMF)
	usage := "6G Session Management Function (SMF)"
	if err := app.RunNfApp("SMF", usage, flags, nf); err != nil {
		fmt.Printf("Failed to run SMF: %+v\n", err)
	}
}
