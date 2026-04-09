package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/pran/service"
	"fmt"
)

func main() {
	nf := new(service.PRAN)
	usage := "6G Proxy RAN Function (PRAN)"
	if err := app.RunNfApp("PRAN", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run PRAN: %+v\n", err)
	}
}
