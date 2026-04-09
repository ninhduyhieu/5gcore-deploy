package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/ausf/service"
	"fmt"
)

func main() {
	nf := new(service.AUSF)
	usage := "6G Authentication Server Network Function (AUSF)"
	if err := app.RunNfApp("AUSF", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run AUSF: %+v\n", err)
	}
}
