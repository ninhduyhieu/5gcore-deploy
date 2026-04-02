package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

type Config struct {
	PlmnId models.PlmnId   `json:"plmnId"`
	Group  string          `json:"group,otmitempty"`
	Mesh   mesh.MeshConfig `json:"mesh"`
}
