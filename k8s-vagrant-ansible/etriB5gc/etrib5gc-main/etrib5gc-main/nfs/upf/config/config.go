package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

// N3 N9 interface
type IfInfo struct {
	Ip   string `json:"ip"`
	Mtu  uint32 `json:"mtu"`
	Name string `json:"name"` //an, tran; known by UPMF
}

type Config struct {
	PlmnId   models.PlmnId   `json:"plmnid"`
	Id       string          `json:"upfid"`
	Slices   []models.Snssai `json:"slices"`
	IfList   []IfInfo        `json:"iflist"`
	GtpPort  *uint16         `json:"gtpPort,omitempty"`
	DevName  string          `json:"devName,omitempty"`
	DnaiList []string        `json:"dnaiList,omitempty"`
	DnnList  []string        `json:"dnnList,omitempty"`
	IsAnchor bool            `json:"isAnchor"`

	Mesh mesh.MeshConfig `json:"mesh"`
}
