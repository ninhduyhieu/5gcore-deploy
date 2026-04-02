package main

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/nfs/udr"
	"fmt"
	"github.com/reogac/sbi/models"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

var _defaultUe UeProfile = UeProfile{
	HomeNetworkPublicKeyId: 1,
	OpType:                 "OPC",
	UacAcc: UacAcc{
		NormalClass: 0,
		Class11:     false,
		Class12:     false,
		Class13:     false,
		Class14:     false,
		Class15:     false,
	},
	UacAic: UacAic{
		Mps: false,
		Mcs: false,
	},
	Integrity: Integrity{
		IA1: true,
		IA2: true,
		IA3: true,
	},
	Ciphering: Ciphering{
		EA1: true,
		EA2: true,
		EA3: true,
	},
	IntegrityMaxRate: IntegrityMaxRate{
		Uplink:   "full",
		Downlink: "full",
	},
	AuthenticationMethod: models.AUTHMETHOD_5G_AKA,
}

//UE subscription data:
// - AuthenticationSubscriptionData
// - AccessAndMobilitySubscriptionData
// - SmfSelectionSubscriptionData
// - SmSubData - map[snssai]SessionManagementSubscriptionData

// for yaml marshaling
type UeProfile struct {
	Supi                   string            `yaml:"supi"`
	Mcc                    string            `yaml:"mcc"`
	Mnc                    string            `yaml:"mnc"`
	ProtectionScheme       int               `yaml:"protectionScheme"`
	HomeNetworkPublicKey   string            `yaml:"homeNetworkPublicKey"`
	HomeNetworkPublicKeyId int               `yaml:"homeNetworkPublicKeyId"`
	RoutingIndicator       string            `yaml:"routingIndicator"`
	Key                    string            `yaml:"key"`
	Op                     string            `yaml:"op"`
	Amf                    string            `yaml:"amf"`
	OpType                 string            `yaml:"opType"`
	Imei                   string            `yaml:"imei"`
	Imeisv                 string            `yaml:"imeisv"`
	GnbSearchList          []string          `yaml:"gnbSearchList"`
	UacAic                 UacAic            `yaml:"uacAic"`
	UacAcc                 UacAcc            `yaml:"uacAcc"`
	Sessions               []Session         `yaml:"sessions"`
	CNssai                 []models.Snssai   `yaml:"configured-snssai"`
	DNssai                 []models.Snssai   `yaml:"default-snssai"`
	Integrity              Integrity         `yaml:"integrity"`
	Ciphering              Ciphering         `yaml:"ciphering"`
	IntegrityMaxRate       IntegrityMaxRate  `yaml:"integrityMaxRate"`
	AuthenticationMethod   models.AuthMethod `yaml:"authenticationMethod"`
}

// build then save UeProfile to a yaml file
func (ue *UeProfile) saveYamlProfile(fn string) (err error) {
	var buf []byte
	var ueCopy UeProfile
	ueCopy = *ue
	ueCopy.Sessions = []Session{}
	ueCopy.CNssai = []models.Snssai{}
	ueCopy.DNssai = []models.Snssai{}

	//add "0x" prefix to all slices
	for _, s := range ue.Sessions {
		if len(s.Slice.Sd) > 0 {
			ueCopy.Sessions = append(ueCopy.Sessions, Session{
				Type:  s.Type,
				Apn:   s.Apn,
				Slice: addSlicePrefix(&s.Slice),
			})
		} else {
			ueCopy.Sessions = append(ueCopy.Sessions, Session{
				Type:  s.Type,
				Apn:   s.Apn,
				Slice: s.Slice,
			})
		}
	}
	for _, s := range ue.CNssai {
		ueCopy.CNssai = append(ueCopy.CNssai, addSlicePrefix(&s))
	}

	for _, s := range ue.DNssai {
		ueCopy.DNssai = append(ueCopy.DNssai, addSlicePrefix(&s))
	}

	//marshaling
	if buf, err = yaml.Marshal(ueCopy); err != nil {
		log.Errorf("error: %+v", err)
		return
	}

	if err = ioutil.WriteFile(fn, buf, 0644); err != nil {
		log.Errorf("error: %+v", err)
		return
	}

	log.Infof("Ue profile is written to %s\n", fn)
	return
}
func (ue *UeProfile) buildAuthSub() *models.AuthenticationSubscription {
	authSub := &models.AuthenticationSubscription{
		AuthenticationManagementField: ue.Amf,
		Supi:                          ue.Supi,
		AuthenticationMethod:          ue.AuthenticationMethod,
		EncPermanentKey:               ue.Key,
		SequenceNumber: &models.SequenceNumber{
			Sqn: "000000000000",
		},
	}
	if ue.OpType == OPC {
		authSub.EncOpcKey = ue.Op
	} else { //OP
		authSub.EncTopcKey = ue.Op
	}
	return authSub
}

func (ue *UeProfile) buildSmSub() *models.SmSubsData {
	subs := []models.SessionManagementSubscriptionData{}
	for _, s := range ue.CNssai {
		subs = append(subs, models.SessionManagementSubscriptionData{
			SingleNssai: s,
			//TODO: add DnnConfigurations
		})
	}

	smSub := &models.SmSubsData{
		SessionManagementSubscriptionData: subs,
	}

	return smSub
}

func (ue *UeProfile) buildAmSub() *models.AccessAndMobilitySubscriptionData {
	nssai := new(models.Nssai)
	for _, s := range ue.CNssai {
		nssai.SingleNssais = append(nssai.SingleNssais, s)
	}

	for _, s := range ue.DNssai {
		nssai.DefaultSingleNssais = append(nssai.DefaultSingleNssais, s)
	}

	amSub := &models.AccessAndMobilitySubscriptionData{
		Nssai: nssai,
		SubscribedUeAmbr: &models.Ambr{
			Uplink:   "1 Gbps",
			Downlink: "2 Gbps",
		},
	}
	return amSub
}

func (ue *UeProfile) buildSmfSel() *models.SmfSelectionSubscriptionData {
	smfSel := &models.SmfSelectionSubscriptionData{}
	return smfSel
}

// save UeModel to backend database
func (ue *UeProfile) saveDb(plmnId *models.PlmnId, db *udr.BackendDb) (err error) {
	authSub := ue.buildAuthSub()
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()
	if err = db.SaveUeAuthSub(ctx1, authSub); err != nil {
		return fmt.Errorf("save authentication subscription data failed: %+v", err)
	}
	smSub := ue.buildSmSub()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err = db.SaveUeSmSub(ctx2, ue.Supi, plmnId, smSub); err != nil {
		return fmt.Errorf("save session management data failed: %+v", err)
	}

	amSub := ue.buildAmSub()
	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()
	if err = db.SaveUeAmSub(ctx3, ue.Supi, plmnId, amSub); err != nil {
		return fmt.Errorf("save access and mobility management data failed: %+v", err)
	}

	smfSel := ue.buildSmfSel()
	ctx4, cancel4 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel4()
	if err = db.SaveUeSmfSel(ctx4, ue.Supi, plmnId, smfSel); err != nil {
		return fmt.Errorf("save Smf selection data failed: %+v", err)
	}
	return
}

//validate slice information
func (ue *UeProfile) rewriteSlices() {
	slices := []models.Snssai{}
	for _, s := range ue.CNssai {
		if newSlice, err := common.RewriteSlice(&s); err == nil {
			slices = append(slices, *newSlice)
		}
	}
	ue.CNssai = slices

	slices = []models.Snssai{}
	for _, s := range ue.DNssai {
		if newSlice, err := common.RewriteSlice(&s); err == nil {
			slices = append(slices, *newSlice)
		}
	}
	ue.DNssai = slices

	sessions := []Session{}
	for _, s := range ue.Sessions {
		if newSlice, err := common.RewriteSlice(&s.Slice); err == nil {
			sessions = append(sessions, Session{
				Type:  s.Type,
				Apn:   s.Apn,
				Slice: *newSlice,
			})
		}
	}
	ue.Sessions = sessions
}

func addSlicePrefix(s *models.Snssai) models.Snssai {
	if len(s.Sd) == 0 {
		return *s
	} else {
		return models.Snssai{
			Sst: s.Sst,
			Sd:  "0x" + s.Sd,
		}
	}
}
