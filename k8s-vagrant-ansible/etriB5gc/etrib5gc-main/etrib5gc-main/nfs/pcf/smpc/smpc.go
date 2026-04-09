package smpc

import (
	"etrib5gc/logctx"
	"etrib5gc/nfs/pcf/context"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/idgen"
	"github.com/sirupsen/logrus"
	"math"
	"sync"
)

var _smpc *SmPolicies
var log *logrus.Entry

func GetSmPolicies() *SmPolicies {
	if _smpc == nil {
		log = logctx.Entry(logrus.Fields{
			"mod": "smpc",
		})
		_smpc = newSmPolicies()
	}
	return _smpc
}

type FlowRule struct {
	precedence float64
	filter     string
	qfi        float64
}

type QosFlow struct {
	mbrUL, mbrDL, gbrUL, gbrDL string
	qfi, _5qi                  float64
}

type SmPolicy struct {
	*logrus.Entry
	id        uint16
	idStr     string
	pccRuleId int32 //pcc rule id generator
	smCtx     *models.SmPolicyContextData
	decision  *models.SmPolicyDecision
	smPolData *models.SmPolicyData //from UDR
}

func (smPol *SmPolicy) Id() string {
	return smPol.idStr
}

func (smPol *SmPolicy) SmPolicyDecision() *models.SmPolicyDecision {
	return smPol.decision
}

func (smPol *SmPolicy) SmPolicyControl() *models.SmPolicyControl {
	return &models.SmPolicyControl{
		Policy:  *smPol.decision,
		Context: *smPol.smCtx,
	}
}

func (smPol *SmPolicy) setId(id uint16) {
	smPol.id = id
	smPol.idStr = fmt.Sprintf("smPol-%d", id)
}

// create sesion rule id from pdu id
func (smPol *SmPolicy) createSessionRuleId() string {
	return fmt.Sprintf("session-rule-%d", smPol.smCtx.PduSessionId) //session rule id
}

// build SmPolicyDecision from SmPolicyContextData and SmPolicyData
func (smPol *SmPolicy) buildDecision() (err error) {
	smPol.Tracef("Build SmPolicyDecision")
	req := smPol.smCtx

	decision := &models.SmPolicyDecision{
		SessRules:     make(map[string]*models.SessionRule),
		PccRules:      make(map[string]*models.PccRule),
		TraffContDecs: make(map[string]*models.TrafficControlData),
		QosDecs:       make(map[string]*models.QosData),
	}
	if req.SubsSessAmbr == nil {
		err = fmt.Errorf("SubsSessAmbr is empty")
		return
	}
	sessRule := models.SessionRule{
		AuthSessAmbr: req.SubsSessAmbr,
		SessRuleId:   smPol.createSessionRuleId(),
		// RefUmData
		// RefCondData
	}
	if req.SubsDefQos == nil {
		err = fmt.Errorf("SubsDefQos is empty")
		return
	}

	defaultQos := req.SubsDefQos
	//just authorized the requested Qos no matter what
	sessRule.AuthDefQos = &models.AuthorizedDefaultQos{
		FiveQi:        &defaultQos.FiveQi,
		Arp:           &defaultQos.Arp,
		PriorityLevel: defaultQos.PriorityLevel,
		// AverWindow
		// MaxDataBurstVol
	}

	decision.SessRules[sessRule.SessRuleId] = &sessRule

	var dnnData *models.SmPolicyDnnData

	if dnnData = smPol.getDnnData(); dnnData == nil { //get SmPolicyDnnData from UDR
		err = fmt.Errorf("SmPolicyDnnData is empty")
		return
	}
	decision.Online = dnnData.Online
	decision.Offline = dnnData.Offline
	decision.Ipv4Index = dnnData.Ipv4Index
	decision.Ipv6Index = dnnData.Ipv6Index

	qosDataList := smPol.buildQosDataList()
	for _, qosData := range qosDataList {
		decision.QosDecs[qosData.QosId] = &qosData
	}

	pccRules := smPol.buildPccRules(decision.QosDecs)
	for _, r := range pccRules {
		decision.PccRules[r.PccRuleId] = r
	}

	decision.SuppFeat = "dummy"
	decision.QosFlowUsage = req.QosFlowUsage
	decision.PolicyCtrlReqTriggers = BuildPolicyTriggers(0x40780f)

	smPol.decision = decision
	return
}

func (smPol *SmPolicy) getDnnData() (dnnData *models.SmPolicyDnnData) {
	//TODO: should be from subscribed data for given Supi/SliceInfo/Dnn
	dnnData = &models.SmPolicyDnnData{ //a dummy one
		Online:    new(bool),
		Offline:   new(bool),
		Ipv4Index: new(int),
		Ipv6Index: new(int),
	}
	*dnnData.Online = true
	return
}

func (smPol *SmPolicy) buildQosDataList() (l []models.QosData) {
	smPol.Tracef("Build QosData list")
	//TODO:should be created with flow information from UDR for specific
	//supi/sliceinfo/dnn
	qosflows := []QosFlow{
		QosFlow{
			qfi:   1,
			_5qi:  9,
			gbrUL: "100 Mbps",
			gbrDL: "100 Mbps",
			mbrUL: "1000 Mbps",
			mbrDL: "1000 Mbps",
		},
	}
	for _, f := range qosflows {
		tmp := models.QosData{
			QosId:   fmt.Sprintf("%d", int(f.qfi)),
			GbrUl:   f.gbrUL,
			GbrDl:   f.gbrDL,
			MaxbrUl: f.mbrUL,
			MaxbrDl: f.mbrDL,
			Qnc:     new(bool),
			FiveQi:  new(int),
		}
		*tmp.FiveQi = int(f._5qi)
		l = append(l, tmp)
	}
	return
}

func (smPol *SmPolicy) generatePccRuleId() string {
	smPol.pccRuleId++
	return fmt.Sprintf("pcc-rule-%d", smPol.pccRuleId)
}

func (smPol *SmPolicy) buildPccRules(qosdecs map[string]*models.QosData) (l []*models.PccRule) {
	smPol.Tracef("Build PccRule list")
	//TODO: get flowrules from UDR
	flowrules := []FlowRule{
		FlowRule{
			filter:     "permit out ip from any to assigned",
			precedence: 10,
			qfi:        1,
		},
	}

	for _, r := range flowrules {
		qfi := fmt.Sprintf("%d", int(r.qfi))
		if _, ok := qosdecs[qfi]; !ok { //qfi not exist, move to next rule
			continue
		}

		flowinfos := []models.FlowInformation{
			models.FlowInformation{
				FlowDescription: r.filter,
				FlowDirection:   models.FLOWDIRECTION_BIDIRECTIONAL,
			},
		}
		tmp := &models.PccRule{
			AppId:      "",
			FlowInfos:  flowinfos,
			PccRuleId:  smPol.generatePccRuleId(),
			Precedence: new(int),
			RefQosData: []string{qfi},
		}
		*tmp.Precedence = int(r.precedence)
		l = append(l, tmp)
	}

	//TODO: add PccRules from TrafficInfluentData (from UDR)
	return
}

func (smPol *SmPolicy) UpdateSmPolicy(req *models.SmPolicyUpdateContextData) (err error) {
	//TODO: update sm policy
	return
}

// SmPolicy pool
type SmPolicies struct {
	smPolIdGen     idgen.Generator[uint16]
	policies       map[string]*SmPolicy //index by smPolId
	sessionIndexes map[string]*SmPolicy //indexed by supi and Pdu session Id
	mutex          sync.RWMutex
}

func newSmPolicies() *SmPolicies {
	return &SmPolicies{
		policies:       make(map[string]*SmPolicy),
		sessionIndexes: make(map[string]*SmPolicy),
		smPolIdGen:     idgen.NewGenerator[uint16](1, math.MaxUint16),
	}
}

func (l *SmPolicies) CreateSmPolicy(req *models.SmPolicyContextData) (rsp *SmPolicy, err error) {
	//delete old one if exists
	l.deleteOldSmPolicy(req.Supi, uint8(req.PduSessionId))

	smPol := &SmPolicy{
		smCtx: req,
		Entry: log.WithFields(logrus.Fields{
			"supi":  req.Supi,
			"pduId": req.PduSessionId,
		}),
	}

	smPol.Tracef("Get SmPollicyData from UDR")
	//get SmPolicyData from UDR
	udr := context.Udr()
	if smPol.smPolData, err = udr.GetSmPolicyData(req.Supi); err != nil { //TODO: may add more parameters
		return
	}

	//create SmPolicyDecision
	if err = smPol.buildDecision(); err != nil {
		return
	}
	//assign a smPol identity
	smPol.setId(l.smPolIdGen.Allocate())
	l.mutex.Lock()
	l.policies[smPol.idStr] = smPol
	l.sessionIndexes[sessionId(req.Supi, uint8(req.PduSessionId))] = smPol
	l.mutex.Unlock()
	rsp = smPol

	log.Infof("Create SmPolicy %s for supi=%s, pduId=%d", smPol.idStr, req.Supi, req.PduSessionId)
	return
}

func (l *SmPolicies) GetSmPolicy(id string) *SmPolicy {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	smPol, _ := l.policies[id]
	return smPol
}

func (l *SmPolicies) delete(smPol *SmPolicy) {
	id := sessionId(smPol.smCtx.Supi, uint8(smPol.smCtx.PduSessionId))
	delete(l.sessionIndexes, id)
	delete(l.policies, smPol.idStr)
	l.smPolIdGen.Free(smPol.id)
	log.Infof("Policy %s deleted", smPol.idStr)
}

func (l *SmPolicies) deleteOldSmPolicy(supi string, pduId uint8) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	id := sessionId(supi, pduId)
	if smPol, ok := l.sessionIndexes[id]; ok {
		l.delete(smPol)
	}
}

func (l *SmPolicies) DeleteSmPolicy(id string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if smPol, ok := l.policies[id]; ok {
		l.delete(smPol)
	}
}
