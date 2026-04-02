package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

type Config struct {
	PlmnId models.PlmnId   `json:"plmnId"`
	Slice  models.Snssai   `json:"slice"`
	Mesh   mesh.MeshConfig `json:"mesh"`
}
