package context

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"etrib5gc/nfs/amf/config"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/nsm/amfman"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	MAX_NUM_UES uint = 10000
)

type SliceMapping struct {
	serving models.Snssai
	isLbo   bool
}

type Plmn struct {
	id     models.PlmnId
	slices map[models.Snssai]SliceMapping //map home slice -> serving slice
}

type AmfContext struct {
	plmnId    models.PlmnId
	amfId     nas.AmfId
	secAlgs   *config.NasSecAlgList
	slices    map[string]models.Snssai
	plmnPeers map[models.PlmnId]Plmn
	ranList   RanList

	t3550           int
	t3555           int
	t3522           int
	t3565           int
	t3513           int
	t3502           uint8
	t3512           uint8
	handoverTimeout int

	maxNumUes uint
}

var _ctx *AmfContext
var log *logrus.Entry

func Init(c *config.Config) (err error) {
	if _ctx != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "context",
	})
	amf := &AmfContext{
		slices:          make(map[string]models.Snssai),
		plmnPeers:       make(map[models.PlmnId]Plmn),
		plmnId:          models.PlmnId(c.PlmnId),
		ranList:         newRanList(),
		secAlgs:         c.Algs,
		t3550:           1000, //milliseconds
		t3555:           1000, //milliseconds
		t3522:           1000,
		t3565:           1000,
		t3513:           1000,
		t3502:           100,
		t3512:           100,
		handoverTimeout: 5000,
		maxNumUes:       MAX_NUM_UES,
	}
	if err = amf.amfId.ParseDecimals(c.AmfSet); err != nil {
		return utils.WrapError("Parse Amf set", err)
	} else {
		log.Tracef("Parse AMFID ok: %s[%s]", amf.amfId.String(), amf.amfId.DecimalString())
	}

	if amf.secAlgs == nil {
		log.Tracef("Amf is using the default algorithm settings")
		amf.secAlgs = &config.DefaultNasSecAlgs
	}
	_ctx = amf
	return
}

func (ctx *AmfContext) requestAmfPointer(nsmCli sbi.ConsumerClient) (err error) {
	ep := mesh.EndpointInfo()
	req := &models.AmfRegistrationRequest{
		PlmnId: ctx.plmnId,
		Uuid:   ep.Uuid,
		AmfSet: AmfSet(),
	}
	var rsp *models.AmfRegistrationResponse
	if rsp, err = amfman.AmfRegister(nsmCli, req); err != nil {
		return
	}
	ctx.amfId.SetPointer(uint8(rsp.AmfPointer))
	for _, s := range rsp.Slices {
		ctx.slices[s.String()] = s
	}
	for _, plmn := range rsp.PlmnPeers {
		ctx.plmnPeers[plmn.Id] = ctx.createPlmn(&plmn)
	}

	log.Infof("Amf is registered with pointer:%s", ctx.amfId.DecimalString())
	return
}

func (ctx *AmfContext) hasSlice(s *models.Snssai) bool {
	_, ok := ctx.slices[s.String()]
	return ok
}

func (ctx *AmfContext) createPlmn(info *models.HomePlmnConfiguration) (plmn Plmn) {
	plmn.id = info.Id
	plmn.slices = make(map[models.Snssai]SliceMapping)
	for _, s := range info.Slices {
		if ctx.hasSlice(&s.ServingSnssai) {
			plmn.slices[s.HomeSnssai] = SliceMapping{
				serving: s.ServingSnssai,
				isLbo:   s.IsLbo,
			}
		}
	}
	return
}

func RegisterAmf(nsmCli sbi.ConsumerClient) (err error) {
	return _ctx.requestAmfPointer(nsmCli)
}

func GetNon3GppDeregistrationTimerValue() uint8 {
	//TODO: get from Amf context
	return 200
}

func Get5gsNwFeatSuppEnable() bool {
	return true
}

func Get5gsNwFeatSuppImsVoPS() uint8 {
	return 1
}
func Get5gsNwFeatSuppEmc() uint8 {
	return 1
}
func Get5gsNwFeatSuppEmf() uint8 {
	return 1
}
func Get5gsNwFeatSuppIwkN26() uint8 {
	return 1
}
func Get5gsNwFeatSuppMpsi() uint8 {
	return 1
}
func Get5gsNwFeatSuppEmcN3() uint8 {
	return 1
}
func Get5gsNwFeatSuppMcsi() uint8 {
	return 1
}

func CheckGuami(guami *models.Guami) bool {
	amfId := _ctx.amfId.String()
	return _ctx.plmnId.Mcc == guami.PlmnId.Mcc && _ctx.plmnId.Mnc == guami.PlmnId.Mnc && guami.AmfId == amfId
}

func NasAmfId() nas.AmfId {
	return _ctx.amfId
}

func AmfSet() string {
	region, set, _ := _ctx.amfId.Get()
	return fmt.Sprintf("%d-%d", region, set)
}

func AmfPointer() uint8 {
	return _ctx.amfId.GetPointer()
}

func NasPlmnId() (id nas.PlmnId) {
	id.Set(_ctx.plmnId.Mcc, _ctx.plmnId.Mnc) //should never fail
	return
}
func PlmnId() *models.PlmnId {
	return &_ctx.plmnId
}

func DefaultDnn(isGpp bool) string {
	//TODO: select a default one from configured list
	return "internet"
}

func T3550() time.Duration {
	return time.Duration(_ctx.t3550) * time.Millisecond
}

func T3555() time.Duration {
	return time.Duration(_ctx.t3555) * time.Millisecond
}

/*
func T3513() int {
	return _ctx.t3513
}
func T3565() int {
	return _ctx.t3565
}
*/
func T3522() time.Duration {
	return time.Duration(_ctx.t3522) * time.Millisecond
}

func T3502() uint8 {
	return _ctx.t3502
}

func T3512() uint8 {
	return _ctx.t3512
}

func HandoverTimeout() int {
	return _ctx.handoverTimeout
}

func MaxNumUes() int {
	return int(_ctx.maxNumUes)
}

func IsServingSlice(s *models.Snssai) bool {
	_, ok := _ctx.slices[s.String()]
	return ok
}

func BuildGuti(tmsi int32) string {
	amfId := _ctx.amfId.String()

	plmnId := _ctx.plmnId.Mcc + _ctx.plmnId.Mnc
	guti := plmnId + amfId + fmt.Sprintf("%08x", tmsi)
	return guti
}

func SelectAlgorithms(ueSecCap *nas.UeSecurityCapability) (encAlg, intAlg uint8) {
	//for oai ue
	encAlg = nas.AlgCiphering128NEA0
	intAlg = nas.AlgIntegrity128NIA2
	supported := false
	for _, alg := range _ctx.secAlgs.IntegrityOrder {
		switch alg {
		case nas.AlgIntegrity128NIA0:
			supported = ueSecCap.GetIA(0)
		case nas.AlgIntegrity128NIA1:
			supported = ueSecCap.GetIA(1)
		case nas.AlgIntegrity128NIA2:
			supported = ueSecCap.GetIA(2)
		case nas.AlgIntegrity128NIA3:
			supported = ueSecCap.GetIA(3)
		}
		if supported {
			intAlg = alg
			break
		}
	}

	supported = false
	for _, alg := range _ctx.secAlgs.CipheringOrder {
		switch alg {
		case nas.AlgCiphering128NEA0:
			supported = ueSecCap.GetEA(0)
		case nas.AlgCiphering128NEA1:
			supported = ueSecCap.GetEA(1)
		case nas.AlgCiphering128NEA2:
			supported = ueSecCap.GetEA(2)
		case nas.AlgCiphering128NEA3:
			supported = ueSecCap.GetEA(3)
		}
		if supported {
			encAlg = alg
			break
		}
	}
	return
}

func GetSliceMapping(plmnId models.PlmnId) map[models.Snssai]models.Snssai {
	if m, ok := _ctx.plmnPeers[plmnId]; ok {
		slices := make(map[models.Snssai]models.Snssai)
		for k, v := range m.slices {
			slices[k] = v.serving
		}
		return slices
	}
	return nil
}

func IsHomeRouted(plmnId models.PlmnId, hSlice models.Snssai) bool {
	if m, ok := _ctx.plmnPeers[plmnId]; ok {
		if serving, ok := m.slices[hSlice]; ok {
			return !serving.isLbo
		}
	}
	return false
}

func GetGuami() models.Guami {
	return models.Guami{
		PlmnId: models.PlmnIdNid{
			Mcc: _ctx.plmnId.Mcc,
			Mnc: _ctx.plmnId.Mnc,
		},
		AmfId: _ctx.amfId.String(),
	}
}

func FullNetworkName() string {
	return "etri6g"
}

func ShortNetworkName() string {
	return "etri"
}
