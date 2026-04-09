package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
)

type NasSecAlgList struct {
	IntegrityOrder []byte `json:"intorder"`
	CipheringOrder []byte `json:"ciporder"`
}

var DefaultNasSecAlgs NasSecAlgList = NasSecAlgList{
	IntegrityOrder: []byte{
		nas.AlgIntegrity128NIA2,
		nas.AlgIntegrity128NIA1,
		nas.AlgIntegrity128NIA3,
		nas.AlgIntegrity128NIA0,
	},
	CipheringOrder: []byte{
		nas.AlgCiphering128NEA0,
		nas.AlgCiphering128NEA1,
		nas.AlgCiphering128NEA2,
		nas.AlgCiphering128NEA3,
	},
}

type Config struct {
	PlmnId models.PlmnId   `json:"plmnId"`
	AmfSet string          `json:"amfSet"`
	Algs   *NasSecAlgList  `json:"algorithm,omitempty"`
	Mesh   mesh.MeshConfig `json:"mesh"`
}
