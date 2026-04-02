package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/udm/service"
	"fmt"
)

func main() {
	nf := new(service.UDM)
	usage := "6G Unified Data Management Function (UDM)"
	if err := app.RunNfApp("UDM", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run UDM: %+v\n", err)
	}
}
