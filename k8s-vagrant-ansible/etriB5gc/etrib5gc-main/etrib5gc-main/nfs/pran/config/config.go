package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

type Config struct {
	TransportNetworks []string          `json:"transportNetworks"` //GnB User Plane interface name
	NgapPort          *int              `json:"ngapPort,omitempty"`
	NgapBinds         []string          `json:"ngapBinds"`
	PlmnId            models.PlmnId     `json:"plmnId"`
	AmfRegion         int               `json:amfRegion`
	Mesh              mesh.MeshConfig   `json:"mesh"`
	NfSelection       map[string]string `json:"nfSelection"`
	NasSplit          bool              `json:"nasSplit,omitempty"`
}
