package common

import "github.com/reogac/sbi/models"

type UpfInfInfo struct {
	Addr string
	Mtu  uint32
}

type UpfInfo struct {
	Slices map[string]models.Snssai
	INFs   map[string]UpfInfInfo
}
