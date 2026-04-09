package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/pcf/service"
	"fmt"
)

func main() {
	nf := new(service.PCF)
	usage := "6G Policy Control Function (PCF)"
	if err := app.RunNfApp("PCF", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run PCF: %+v\n", err)
	}
}
