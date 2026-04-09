package sm

import (
	"context"
	"etrib5gc/internal/fsm"
	"fmt"
	"github.com/reogac/sbi/models"
	"net/http"

	"github.com/reogac/nas"
)

func (smCtx *SmContext) HandlePostSmContext(req *models.PostSmContextsRequest) (headers map[string]string, rsp *models.PostSmContextsResponse, errRsp *models.PostSmContextsErrorResponse) {
	var err error

	//1. get SM data subscription from UDM
	if err = smCtx.getSmData(); err != nil {
		smCtx.Errorf("Fail to get SmData: %+v", err)
		errRsp = &models.PostSmContextsErrorResponse{
			JsonData: &models.SmContextCreateError{
				Error: models.ExtProblemDetails{
					Status: http.StatusInternalServerError,
					Detail: "Fail to get SmData",
				},
			},
		}
		return
	} else {
		smCtx.Infof("Retrieved session management subscription data from UDM")
	}

	//if AMF send N1SM (PduSessionEstablishmentRequest), we will only create a
	//new SmContext if the message is valid.

	//2. decode Nas Sm message
	var nasMsg nas.NasMessage
	if nasMsg, err = nas.Decode(nil, req.BinaryDataN1SmMessage, true); err == nil {
		if gsm := nasMsg.Gsm; gsm == nil {
			err = fmt.Errorf("N1Sm decoding fail: no Gsm message found")
		} else if gsm.MsgType != nas.PduSessionEstablishmentRequestMsgType {
			err = fmt.Errorf("Expecting PduSessionEstablishmentRequest")
		}
	}

	if err != nil {
		smCtx.Errorf("Invalid N1Sm message: %+v", err)
		errRsp = &models.PostSmContextsErrorResponse{
			JsonData: &models.SmContextCreateError{
				Error: models.ExtProblemDetails{
					Status: http.StatusInternalServerError,
					Detail: "Invalid N1Sm",
				},
			},
		}
		if pdu, err := smCtx.buildGsmStatus(nas.PtiUnspecified, 0); err != nil {
			smCtx.Errorf("Fail to build 5GsmStatus: %+v", err)
		} else {
			errRsp.BinaryDataN1SmMessage = pdu
		}
		return
	}

	//3. send N1Sm to state machine
	smCtx.sendEvent(context.Background(), fsm.NewEventData(N1SmEvent, nasMsg.Gsm))

	//4. add session context to the pool
	_pool.add(smCtx)

	//5. prepare a response
	headers = make(map[string]string)
	headers["Location"] = smCtx.idString()

	rsp = &models.PostSmContextsResponse{
		JsonData: &models.SmContextCreatedData{
			SmContextRef: smCtx.idString(),
			UpCnxState:   smCtx.getUpCnxState(),
		},
	}

	return
}
