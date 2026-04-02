package main

import "github.com/reogac/sbi/models"

type Session struct {
	Type  string        `json:"type"`
	Apn   string        `jons:"apn"`
	Slice models.Snssai `json:"slice"`
}

type UacAic struct {
	Mps bool `json:"mps"`
	Mcs bool `json:"mcs"`
}

type UacAcc struct {
	NormalClass int32 `json:"normalClass" yaml:"normalClass"`
	Class11     bool  `json:"class11"`
	Class12     bool  `json:"class12"`
	Class13     bool  `json:"class13"`
	Class14     bool  `json:"class14"`
	Class15     bool  `json:"class15"`
}

type Integrity struct {
	IA1 bool `json:"IA1" yaml:"IA1"`
	IA2 bool `json:"IA2" yaml:"IA2"`
	IA3 bool `json:"IA3" yaml:"IA3"`
}

type Ciphering struct {
	EA1 bool `json:"EA1" yaml:"EA1"`
	EA2 bool `json:"EA2" yaml:"EA2"`
	EA3 bool `json:"EA3" yaml:"EA3"`
}

type IntegrityMaxRate struct {
	Uplink   string `json:"uplink"`
	Downlink string `json:"downlink"`
}
