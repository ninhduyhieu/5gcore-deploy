package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/damf/service"
	"fmt"
)

func main() {
	nf := new(service.DAMF)
	usage := "6G Default Access and Mobility Management Function (DAMF)"
	if err := app.RunNfApp("DAMF", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run DAMF: %+v\n", err)
	}
}
