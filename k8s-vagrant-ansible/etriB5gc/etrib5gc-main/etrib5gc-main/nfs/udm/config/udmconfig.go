package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

type Config struct {
	PlmnId models.PlmnId   `json:"plmnid"`
	Mesh   mesh.MeshConfig `json:"mesh"`
}
