package context

import (
	"etrib5gc/logctx"
	"etrib5gc/nfs/pran/config"
	"fmt"

	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/nsm/amfman"
	"github.com/reogac/sbi/models"
)

const (
	DUMB_AMF_ID   = "0a1900"
	DUMB_AMF_NAME = "AMF"

	MAX_NUM_UES uint = 10000
)

var _cu *CuContext

type CuContext struct {
	logctx.LogWriter
	transportNetworks []string                   //list of gnB transport networks
	nfSelection       map[string]string          //options for NF selection
	amfName           string                     //dummy AMF name to send to gnB
	amfId             string                     //dummy AmfId to send to gnB
	plmnId            models.PlmnId              //serving Plmn
	slices            map[string]models.Snssai   //list of slices supported by AMFs connected to PRAN
	tacs              map[string][]models.Snssai //supported TACs and slices
	relCap            int64
	closed            int32 //indicator if PRAN is terminated
	maxNumUes         uint
	nasSplit          bool
	amfRegion         uint8
	subscribedAmfs    map[string]Amf
}

func Init(c *config.Config) {
	if _cu == nil {
		_cu = &CuContext{
			LogWriter: logctx.WithFields(logctx.Fields{
				"mod": "context",
			}),
			transportNetworks: c.TransportNetworks,
			amfName:           DUMB_AMF_NAME,
			amfId:             DUMB_AMF_ID,
			amfRegion:         uint8(c.AmfRegion),
			plmnId:            models.PlmnId(c.PlmnId),
			slices:            make(map[string]models.Snssai),
			tacs:              make(map[string][]models.Snssai),
			subscribedAmfs:    make(map[string]Amf),
			closed:            0,
			nfSelection:       make(map[string]string),
			nasSplit:          c.NasSplit,
			relCap:            255,
			maxNumUes:         MAX_NUM_UES,
		}

		for k, v := range c.NfSelection {
			_cu.nfSelection[k] = v
		}

		//create a dummy Plmn list to satisfy gnB in ngap establishment
		/*
			ctx.plmnList[ctx.plmnId] = []models.Snssai{
				models.Snssai{
					Sst: 1,
					Sd:  "010203",
				},
				models.Snssai{
					Sst: 1,
					Sd:  "000001",
				},
			}
		*/
	}
}

func CheckSupportedRan(plmnId models.PlmnId) bool {
	if plmnId.Mcc != _cu.plmnId.Mcc || plmnId.Mnc != _cu.plmnId.Mnc {
		_cu.Errorf("Ran from different network not accepted")
		return false
	}

	return true
}

func RanNets() []string {
	return _cu.transportNetworks
}

// get dummy AmfId to send to RAN
func AmfId() string {
	return _cu.amfId
}

func AmfRegion() uint8 {
	return _cu.amfRegion
}

func AmfName() string {
	return _cu.amfName
}
func GetNfSelection() (options map[string]string) {
	options = make(map[string]string)
	for k, v := range _cu.nfSelection {
		options[k] = v
	}
	return
}
func NasSplit() bool {
	return _cu.nasSplit
}

func MaxNumUes() int {
	return int(_cu.maxNumUes)
}
func PlmnId() *models.PlmnId {
	return &_cu.plmnId
}

func RelativeCapacity() int64 {
	return _cu.relCap
}

//request NSM to get supported Plmn list for amf region
func GetSupportedSlices(cli sbi.ConsumerClient) (err error) {
	var rsp *models.GetSupportedSlicesResponse
	if rsp, err = amfman.GetSupportedSlices(cli, &models.GetSupportedSlicesRequest{
		AmfRegion: fmt.Sprintf("%d", _cu.amfRegion),
	}); err == nil {
		_cu.Infof("Get Supported Slices : %+v", rsp.Slices)
		for _, item := range rsp.Slices {
			_cu.slices[item.String()] = item
		}
		return
	}
	return
}
