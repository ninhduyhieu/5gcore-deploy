package main

import (
	"etrib5gc/nfs/amf/service"
	"etrib5gc/nfs/app"
	"fmt"
)

func main() {
	nf := new(service.AMF)
	usage := "6G Access and Mobility Management Function (AMF)"
	if err := app.RunNfApp("AMF", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run AMF: %+v\n", err)
	}
}
