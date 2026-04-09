package dataplane

import (
	"github.com/reogac/pfcp/ie"
	"time"
)

const (
	RULE_INIT RuleState = iota
	RULE_CREATED
	RULE_UPDATE
	RULE_DELETE
)

type RuleState uint8

// Forwarding Parameters.
type ForwardingParameters struct {
	DestinationInterface uint8
	NetworkInstance      string
	OuterHeaderCreation  *ie.OuterHeaderCreation
	//	face                 GtpFace
	ForwardingPolicyID string
	SendEndMarker      bool
}

type FAR struct {
	id        uint32
	action    *ie.ApplyAction
	fwdParams *ForwardingParameters
	bar       *BAR
	state     RuleState
}

func (far *FAR) toCreateFar() ie.CreateFAR {
	cFar := ie.CreateFAR{
		FARID: far.id,
	}
	if far.action != nil {
		cFar.ApplyAction = *far.action // ApplyAction_DROP??	////////////////
	}
	if far.bar != nil {
		cFar.BARID = &far.bar.id
	}
	if far.fwdParams != nil {
		cFar.ForwardingParameters = ie.NewForwardingParameters(
			far.fwdParams.DestinationInterface,
			far.fwdParams.NetworkInstance,
			far.fwdParams.OuterHeaderCreation,
			far.fwdParams.ForwardingPolicyID,
		)
	}
	return cFar
}

func (far *FAR) toUpdateFar() ie.UpdateFAR {
	var flag uint8
	if far.fwdParams.SendEndMarker {
		flag = PFCPSMReqFlags_SNDEM
	}
	uFar := ie.UpdateFAR{
		FARID:       far.id,
		ApplyAction: far.action, // ApplyAction_DROP??
	}
	if far.bar != nil {
		uFar.BARID = &far.bar.id
	}
	if far.fwdParams != nil {
		uFar.UpdateForwardingparameters = ie.NewUpdateForwardingParameters(
			far.fwdParams.DestinationInterface,
			far.fwdParams.NetworkInstance,
			far.fwdParams.OuterHeaderCreation,
			far.fwdParams.ForwardingPolicyID,
			flag,
		)
	}
	return uFar
}

type BAR struct {
	id                             uint8
	downlinkDataNotificationDelay  time.Duration
	suggestedBufferingPacketsCount uint8
	state                          RuleState
}

func (bar *BAR) toCreateBar() *ie.CreateBAR {
	delay := uint8(bar.downlinkDataNotificationDelay.Milliseconds())
	cBar := &ie.CreateBAR{
		BARID:                         bar.id,
		DownlinkDataNotificationDelay: &delay,
	}
	return cBar
}

type URR struct {
	id                     uint32
	measureMethod          string // vol or time
	reportingTrigger       uint32
	measurementPeriod      time.Duration
	measurementInformation *ie.MeasurementInformation
	volumeThreshold        *ie.VolumeThreshold
	state                  RuleState
}

func (urr *URR) toCreateUrr() ie.CreateURR { ////////////////////////
	period := uint32(urr.measurementPeriod / time.Second)
	cUrr := ie.CreateURR{
		URRID:             urr.id,
		ReportingTriggers: urr.reportingTrigger,
		MeasurementPeriod: &period,
	}
	if urr.volumeThreshold != nil {
		cUrr.VolumeThreshold = urr.volumeThreshold
	}
	if urr.measurementInformation != nil {
		cUrr.MeasurementInformation = urr.measurementInformation
	}
	switch urr.measureMethod {
	case MesureMethodVol:
		cUrr.MeasurementMethod = ie.NewMeasurementMethod(1, 1, 0)
	case MesureMethodTime:
		cUrr.MeasurementMethod = ie.NewMeasurementMethod(1, 0, 1)
	}
	return cUrr
}

type MAR struct {
	//TODO
	state RuleState
}
