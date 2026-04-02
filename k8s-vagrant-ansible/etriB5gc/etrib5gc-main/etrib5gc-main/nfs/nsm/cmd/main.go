package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/nsm/service"
	"fmt"
)

func main() {
	nf := new(service.NSM)
	usage := "6G Resource Management Function (RMF)"
	if err := app.RunNfApp("RMF", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run RMF: %+v\n", err)
	}
}
