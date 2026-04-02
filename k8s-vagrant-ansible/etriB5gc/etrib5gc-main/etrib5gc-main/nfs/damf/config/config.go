package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

type Config struct {
	PlmnId models.PlmnId   `json:"plmnid"`
	T3502  uint8           `json:"t3502,omitempty"`
	T3560  uint16          `json:"t3560,omitempty"`
	Mesh   mesh.MeshConfig `json:"mesh"`
}
