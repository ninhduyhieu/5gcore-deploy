package config

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi/models"
)

type PlmnConfig struct {
	PlmnId models.PlmnId        `json:"plmnId"`
	Slices []SliceMappingConfig `json:"slices"`
	Sepps  []string             `json:"sepps"`
}

type SliceMappingConfig struct {
	Home    models.Snssai `json:"home"`
	Serving models.Snssai `json:"serving"`
	IsLbo   bool          `json:"isLbo"` //local break out roaming
}

type SliceConfig struct {
	Id           models.Snssai `json:"id"`
	DataNetworks []string      `json:"dataNetworks"` //list of data network supported by this slice
}

type AmfSetConfig struct {
	SetId  string          `json:"setId"`
	Slices []models.Snssai `json:"slices"`
}

type ServerIp struct {
	V4 string `json:"v4,omitempty"`
	V6 string `json:"v6,omitempty"`
}

type DataNetworkConfig struct {
	Name       string    `json:"name"`                 //data network name
	Cidr       string    `json:"cidr"`                 //ip domain
	DhcpServer string    `json:"dhcpServer,omitempty"` //dhcp server (if Smf not to allocate IP)
	NumIpPools int       `json:"numIpPools"`           //number of Ue Ip pools
	Dns        *ServerIp `json:"dns,omitempty"`
	Pcscf      *ServerIp `json:"pcscf,omitempty"`
}

type Config struct {
	PlmnId            models.PlmnId           `json:"plmnid"`
	PlmnPeers         []PlmnConfig            `json:"plmnPeers"` //roaming home networks
	Mesh              mesh.MeshConfig         `json:"mesh"`
	DataNetworks      []DataNetworkConfig     `json:"dataNetworks"`      //data network definitions
	TransportNetworks []string                `json:"transportNetworks"` //list of transport network names
	Slices            []SliceConfig           `json:"slices"`            //list of deployed slices
	AmfSets           []AmfSetConfig          `json:"amfSets"`           //amf set definitions
	Udr               models.UdrConfiguration `json:"udr"`               //udr mongodb configuration
	SuciProfiles      []models.SuciProfile    `json:"suciProfiles"`      //list of protection schemes for SUCI
}
