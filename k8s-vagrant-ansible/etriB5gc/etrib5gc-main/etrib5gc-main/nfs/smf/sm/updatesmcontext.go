package sm

import (
	"etrib5gc/internal/eventmux"
	"etrib5gc/internal/fsm"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"strings"
)

type UpdateSmContextContext struct {
	Request     *models.UpdateSmContextRequest
	Response    *models.UpdateSmContextResponse
	ErrResponse *models.UpdateSmContextErrorResponse
	*eventmux.AsyncTask
}

func NewUpdateSmContextContext(req *models.UpdateSmContextRequest) *UpdateSmContextContext {
	return &UpdateSmContextContext{
		Request:   req,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

// try to intepret the intention of the UpdateSmContext request.
// this is where the 3GPP sucks. Having a pile of shit in a single request to
// cover many actions is stupid. But anyway, we have to deal with that.

func (smCtx *SmContext) handleUpdateSmContextRequest(task *UpdateSmContextContext) {
	req := task.Request
	dat := req.JsonData
	if dat == nil {
		task.Finalize(fmt.Errorf("Empty SmContextUpdateData in UpdateSmContextRequest "))
		return
	}

	//set pending UpdateSmContext task for sending error or N1N2Messages
	smCtx.n1n2.attachPending(task)

	var err error
	n2SmInfo := req.BinaryDataN2SmInformation //HandoverRequestAcknowledgeTransfer

	if len(dat.HoState) > 0 { //handover (NOTE: any handover related N2SmInfo message should be handled here)
		smCtx.Debugf("Handle UpdateSmContext with HoState")
		switch dat.HoState {

		case models.HOSTATE_PREPARING: //HandoverPreparing has a N2SmInfo binary with N2SmInfoType
			err = smCtx.handleHoStatePreparing(dat, n2SmInfo)

		case models.HOSTATE_PREPARED:
			err = smCtx.handleHoStatePrepared(dat, n2SmInfo)

		case models.HOSTATE_COMPLETED:
			err = smCtx.handleHoStateCompleted()

		case models.HOSTATE_CANCELLED:
			err = smCtx.handleHoStateCancelled()

		default:
			err = fmt.Errorf("Invalid HoState in UpdateSmContextRequest")
		}
	} else if len(dat.UpCnxState) > 0 { //activate or deactive user plane
		smCtx.Debugf("Handle UpdateSmContext with UpCnxState")
		err = smCtx.handleUpCnxState(dat.UpCnxState)
	} else if len(dat.Cause) > 0 { //AMF request to remove a duplicate session
		smCtx.Debugf("Handle UpdateSmContext with Cause")
		if strings.Compare(string(req.JsonData.Cause), string(models.CAUSE_REL_DUE_TO_DUPLICATE_SESSION_ID)) == 0 {
			smCtx.Infof("AMF request to release the SmContext due to a duplication")
			event := fsm.NewEventData(ReleaseTriggerEvent, &EventData{
				evType: RELEASE_AMF_DUP,
			})
			smCtx.state.SetNextEvent(event)
		} else { //cause not process
			err = fmt.Errorf("Cause[%s] from UpdateSmContextRequest is not handled", req.JsonData.Cause)
		}
	} else { //might have N1Sm and N2SmInfo
		if len(req.BinaryDataN1SmMessage) > 0 { //has N1Sm message
			smCtx.Debugf("Handle UpdateSmContext with N1Sm")
			//decode N1Sm first
			var nasMsg nas.NasMessage
			if nasMsg, err = nas.Decode(nil, req.BinaryDataN1SmMessage, true); err == nil {
				if nasMsg.Gsm == nil {
					err = fmt.Errorf("No N1Sm message found after decoding from UpdateSmContextRequest ")
				}
			} else {
				err = utils.WrapError("Decode N1Sm from UpdateSmContextRequest", err)
			}

			if err != nil {
				smCtx.Errorf("Invalid N1SM: %+v", err)
				smCtx.sendGsmStatus(nas.PtiUnspecified, nas.Cause5GSMSemanticallyIncorrectMessage)
				err = nil
			} else {
				//otherwise, handle N1Sm
				smCtx.handleN1Sm(nasMsg.Gsm)
			}
		}
		if len(dat.N2SmInfoType) > 0 {
			smCtx.Debugf("Handle UpdateSmContext with N2SmInfo")
			//For PathSwith, we need to update RanUeInfo; otherwise just need to
			//handle the N2SmInfo message
			switch dat.N2SmInfoType {
			case models.N2SMINFOTYPE_PATH_SWITCH_REQ:
				err = smCtx.handlePathSwitchRequest(&smContextUpdateData{dat}, n2SmInfo)
			case models.N2SMINFOTYPE_PATH_SWITCH_SETUP_FAIL:
				err = smCtx.handlePathSwitchSetupFail(&smContextUpdateData{dat}, n2SmInfo)
			default:
				err = smCtx.handleN2SmInfo(dat.N2SmInfoType, n2SmInfo)
			}
		}
	}
	//make sure to let caller know that the task is handled
	task.Finalize(err)
	return
}
