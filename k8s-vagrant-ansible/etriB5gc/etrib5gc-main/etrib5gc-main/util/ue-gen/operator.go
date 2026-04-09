package main

import (
	"etrib5gc/common"
	"etrib5gc/util/suci"
	"fmt"
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"strings"
)

type SliceConfig struct {
	Id      models.Snssai `json:"id"`
	DnnList []string      `json:"dnnlist"`
}

type OperatorConfig struct {
	Db       models.UdrConfiguration `json:"udr"`
	PlmnId   models.PlmnId           `json:"plmnId"`
	Profiles []suci.Profile          `json:"profiles"`
	Amf      string                  `json:"amf"`
	Slices   []SliceConfig           `json:"slices"`
}

type Operator struct {
	config *OperatorConfig
}

func newOperator(cfg *OperatorConfig) *Operator {
	//validate slices
	slices := []SliceConfig{}

	for _, s := range cfg.Slices {
		if newSlice, err := common.RewriteSlice(&s.Id); err == nil {
			slices = append(slices, SliceConfig{
				Id:      *newSlice,
				DnnList: s.DnnList,
			})
		}
	}
	cfg.Slices = slices
	log.Infof("Slices: %v", cfg.Slices)
	return &Operator{
		config: cfg,
	}
}

// generate an UE
func (o *Operator) generateUe(zero *UeProfile, inc bool, supiGen func() string) *UeProfile {
	ue := zero
	if !inc {
		ue.Key = o.randUeKey()
		ue.Op = o.randOp()
	}
	ue.Supi = supiGen()
	ue.Imei = o.randImei()
	//ue.Imeisv = o.randImeiSv() //no IMEISV
	return ue
}

func (o *Operator) isValidSlice(s *models.Snssai) bool {
	for _, i := range o.config.Slices {
		if common.IsSliceEqual(&i.Id, s) {
			return true
		}
	}
	return false
}

func (o *Operator) isValidSession(s *Session) bool {
	if s.Type != "IPv4" {
		log.Warnf("Unsupported session type %s", s.Type)
		return false
	}
	for _, i := range o.config.Slices {
		if common.IsSliceEqual(&i.Id, &s.Slice) {
			for _, apn := range i.DnnList {
				if apn == s.Apn {
					return true
				}
			}
			log.Warnf("Slice %d-%s does not have %s APN", i.Id.Sst, i.Id.Sd, s.Apn)
			return false
		}
	}

	log.Warnf("Slice %d-%s not supported", s.Slice.Sst, s.Slice.Sd)
	return false
}

// create base UE profile
func (o *Operator) copyTemplate(tmpl *UeProfile, share bool) (ue *UeProfile) {
	ue = &UeProfile{
		Mcc:                    o.config.PlmnId.Mcc,
		Mnc:                    o.config.PlmnId.Mnc,
		HomeNetworkPublicKeyId: tmpl.HomeNetworkPublicKeyId,
		RoutingIndicator:       tmpl.RoutingIndicator,
		OpType:                 tmpl.OpType,
		Amf:                    o.config.Amf,
		GnbSearchList:          tmpl.GnbSearchList,
		UacAic:                 tmpl.UacAic,
		UacAcc:                 tmpl.UacAcc,
		Integrity:              tmpl.Integrity,
		Ciphering:              tmpl.Ciphering,
		IntegrityMaxRate:       tmpl.IntegrityMaxRate,
		AuthenticationMethod:   tmpl.AuthenticationMethod,
	}

	if len(ue.RoutingIndicator) == 0 {
		ue.RoutingIndicator = DEFAULT_ROUTING_INDICATOR
	}

	if len(ue.GnbSearchList) == 0 {
		ue.GnbSearchList = []string{"127.0.0.1"}
	}

	if keyInd := ue.HomeNetworkPublicKeyId; keyInd > len(o.config.Profiles) || keyInd < 1 { //invalid value
		ue.HomeNetworkPublicKeyId = 0
		ue.ProtectionScheme = int(nas.ProtectionSchemeNullScheme)
		ue.HomeNetworkPublicKey = ""
	} else {
		ue.ProtectionScheme = o.config.Profiles[keyInd-1].ProtectionScheme
		ue.HomeNetworkPublicKey = o.config.Profiles[keyInd-1].PublicKey
		ue.HomeNetworkPublicKeyId = keyInd
	}

	if share {
		ue.Key = o.randUeKey()
		ue.Op = o.randOp()
	}

	switch ue.AuthenticationMethod {
	case models.AUTHMETHOD_5G_AKA:
	case models.AUTHMETHOD_EAP_AKA_PRIME:
	default:
		ue.AuthenticationMethod = models.AUTHMETHOD_5G_AKA
	}

	switch strings.ToUpper(ue.OpType) {
	case OP:
		ue.OpType = OP
	case OPC:
		ue.OpType = OPC
	default:
		ue.OpType = OPC
	}
	configuredSlices := make(map[models.Snssai]bool)
	for _, s := range tmpl.CNssai {
		if o.isValidSlice(&s) {
			configuredSlices[s] = false
		}
	}
	for _, s := range tmpl.DNssai {
		if o.isValidSlice(&s) {
			configuredSlices[s] = true
		}
	}

	if len(configuredSlices) == 0 {
		for i, s := range o.config.Slices {
			if i == 0 {
				configuredSlices[s.Id] = true //default slice
			} else {
				configuredSlices[s.Id] = false
			}
		}
	}

	ue.CNssai = []models.Snssai{}
	ue.DNssai = []models.Snssai{}
	for s, f := range configuredSlices {
		ue.CNssai = append(ue.CNssai, s)
		if f {
			ue.DNssai = append(ue.DNssai, s)
		}
	}

	sessions := []Session{}
	for _, s := range tmpl.Sessions {
		if o.isValidSession(&s) {
			if _, ok := configuredSlices[s.Slice]; ok {
				sessions = append(sessions, s)
			}
		}
	}
	ue.Sessions = sessions
	return
}

func (o *Operator) randUeKey() (key string) {
	//generate a random key for UE
	key = md5Hash(randSeq(16))
	return
}

func (o *Operator) randOp() (key string) {
	//generate a random key for UE
	key = md5Hash(randSeq(16))
	return
}

func (o *Operator) makeIncSupiGen(firstImsi int) func() string {
	counter := firstImsi - 1
	if counter < 0 {
		counter = 0
	}

	prefix := o.config.PlmnId.Mcc + o.config.PlmnId.Mnc
	suffixLen := 15 - len(prefix)
	formatStr := fmt.Sprintf("imsi-%s%%0%dd", prefix, suffixLen)

	return func() string {
		counter++
		return fmt.Sprintf(formatStr, counter)
	}
}

func (o *Operator) randSupi() (supi string) {
	//generate a random supi
	mcclen := len(o.config.PlmnId.Mcc)
	mnclen := len(o.config.PlmnId.Mnc)
	msisdnlen := 15 - mcclen - mnclen
	prefix := o.config.PlmnId.Mcc + o.config.PlmnId.Mnc
	msisdn := generateRandomMsisdn(msisdnlen)
	supi = "imsi-" + prefix + msisdn
	return
}

func (o *Operator) randImei() (imei string) {
	prefix := "35693803" // model N900 Nokia
	imei = goluhn.GenerateWithPrefix(prefix, 15)
	return
}
