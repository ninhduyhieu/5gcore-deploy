package main

import (
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/sepp"
	"fmt"
)

func main() {
	nf := new(sepp.SEPP)
	usage := "6G SEPP"
	if err := app.RunNfApp("SEPP", usage, nil, nf); err != nil {
		fmt.Printf("Failed to run SEPP: %+v\n", err)
	}
}
