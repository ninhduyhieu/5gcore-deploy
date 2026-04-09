package main

import (
	"etrib5gc/mesh/controller"
	"etrib5gc/nfs/app"
	"fmt"
)

func main() {
	nf := new(controller.Controller)
	usage := "6G Core Network Service Controller"
	if err := app.RunApp("CTRL", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run CTRL: %+v\n", err)
	}
}
