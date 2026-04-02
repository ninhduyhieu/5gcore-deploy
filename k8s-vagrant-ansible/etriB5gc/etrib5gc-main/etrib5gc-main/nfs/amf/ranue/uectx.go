package ranue

import (
	"github.com/reogac/sbi/apis/pran/uectx"
	"github.com/reogac/sbi/models"
)

// send release command to PRAN then release RanUe
func (ranUe *RanUe) ReleaseContext(cause models.N2Cause) (*models.UeContextReleaseComplete, error) {
	//mark RanUe as released so that it can reject any NAS messages
	ranUe.isReleased = true

	// send UeContextReleaseCommand to PRAN
	msg := &models.UeContextReleaseCommand{
		Cause: cause,
		//TODO: add paging priority
	}
	rsp, err := uectx.UeContextRelease(ranUe.ranCli, ranUe.ranUeId.Id, msg)
	if err == nil {
		// deactivate sessions
		ranUe.Infof("UeContextReleaseCommand was sent to gnB, receive a UeContextReleaseResponse")
	}
	_ranUePool.remove(ranUe)
	return rsp, err
}

//gnb request for release UeContext
func (ranUe *RanUe) ReceiveUeContextReleaseRequest(msg *models.UeContextReleaseRequest) error {
	ranUe.Debugf("Handle UeContext Release Request")
	//TODO: send to worker pool
	go func() {
		ranUe.ue.ReceiveUeContextReleaseRequest(ranUe.isGpp, msg)
	}()

	return nil
}

func (ranUe *RanUe) ReceiveRrcInactivityStatusReport(msg *models.RrcInactivityTransportReport) error {
	return nil
}

func (ranUe *RanUe) SendInitialContextSetupRequest(req *models.UeContextSetupRequest) (*models.UeContextSetupResponse, *models.UeContextSetupFailure, error) {
	req.AmfUeInfo = ranUe.buildAmfUeInfo()
	return uectx.UeContextSetup(ranUe.ranCli, ranUe.ranUeId.Id, req)
}
