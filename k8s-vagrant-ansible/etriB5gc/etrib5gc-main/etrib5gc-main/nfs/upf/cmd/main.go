package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/upf/service"
	"fmt"
)

func main() {
	nf := new(service.UPF)
	usage := "6G User Plane Function (UPF)"
	if err := app.RunNfApp("UPF", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run UPF: %+v\n", err)
	}
}
