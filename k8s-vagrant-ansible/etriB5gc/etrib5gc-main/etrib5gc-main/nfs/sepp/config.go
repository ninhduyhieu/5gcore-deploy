package sepp

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

type Config struct {
	PlmnId models.PlmnId   `json:"plmnId"`
	Bind   string          `json:"bind"`
	Url    string          `json:"url"`
	Mesh   mesh.MeshConfig `json:"mesh"`
}
