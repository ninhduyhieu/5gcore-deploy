package sm

import (
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/amf/comm"
	"github.com/reogac/sbi/apis/pran/n1n2dl"
	"github.com/reogac/sbi/apis/pran/nasdl"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"sync"
	"time"
)

const (
	N1N2_TIMEOUT time.Duration = time.Second
)

type N1N2Pending struct {
	id     int64
	doneCh chan error
}

type N1N2Sender struct {
	smCtx    *SmContext
	ranCli   sbi.ConsumerClient
	ranUeId  *models.RanUeId
	pending  *UpdateSmContextContext
	n1n2List map[int64]*N1N2Pending
	idGen    int64 //generate id for pending N1N2
	mu       sync.Mutex
}

func newN1N2Sender(smCtx *SmContext) *N1N2Sender {
	return &N1N2Sender{
		smCtx:    smCtx,
		n1n2List: make(map[int64]*N1N2Pending),
	}
}

func (s *N1N2Sender) setRanUeContext(info *models.RanUeInfo) error {
	s.smCtx.Tracef("Create  RAN consumer [%s]", models.EndpointInfoToString(info.RanInfo))
	var err error
	if s.ranCli, err = mesh.ConsumerFromEndpoint(&info.RanInfo); err != nil {
		return utils.WrapError("Create RAN consumer", err)
	}
	s.ranUeId = &info.RanUeId
	return nil
}

// attach a pending UpdateSmContext task for sending N1/N2 messages in response
// NOTE: call Finalize or sendN1N2 to consume this
// pending task
func (s *N1N2Sender) attachPending(t *UpdateSmContextContext) {
	s.mu.Lock()
	s.pending = t
	t.SetFinalizer(func(fn func(error)) func(error) {
		return func(err error) {
			s.mu.Lock()
			s.pending = nil
			s.mu.Unlock()
			fn(err)
		}
	})
	s.mu.Unlock()
}

/*
func (s *N1N2Sender) WriteError(err error) {
	s.mu.Lock()
	if s.pending != nil {
		ersp := &models.UpdateSmContextErrorResponse{
			JsonData: new(models.SmContextUpdateError),
		}
		ersp.JsonData.UpCnxState = s.smCtx.getUpCnxState()
		ersp.JsonData.Error = models.ExtProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		}
		s.pending.ErrResponse = ersp
	} else {
		s.smCtx.Errorf(err.Error())
	}
	s.mu.Unlock()
}
*/

func (s *N1N2Sender) sendN1N2(msg *N1N2Messages) error {
	if s.ranCli != nil {
		//send directly o PRAN
		if len(msg.n2SmInfo) == 0 { //send N1 only
			if err := nasdl.NasDl(s.ranCli, s.ranUeId.Id, &models.NasDownlinkTransport{
				NasPdu: msg.n1Sm,
			}); err != nil {
				return utils.WrapError("Send Nas downlink", err)
			}

		} else { //send N1 piggybacked on a service that send N2SmInfo
			switch msg.n2SmInfoType {

			case models.N2SMINFOTYPE_PDU_RES_REL_CMD:
				req := &models.SessionResourceReleaseRequest{
					SessionId: int16(s.smCtx.pduSessionId),
					Transfer:  msg.n2SmInfo,
					N1Sm:      msg.n1Sm,
				}
				if rsp, err := n1n2dl.SessionResourceRelease(s.ranCli, s.ranUeId.Id, req); err == nil {
					//handle response
					s.smCtx.handleN2SmInfo(models.N2SMINFOTYPE_PDU_RES_REL_RSP, rsp.Transfer)
				} else {
					return utils.WrapError("Request SessionResourceRelease", err)
				}

			case models.N2SMINFOTYPE_PDU_RES_SETUP_REQ:
				req := &models.SessionResourceSetupRequest{
					Ref:       s.smCtx.idString(),
					Smf:       *mesh.EndpointInfo(),
					Snssai:    *s.smCtx.snssai,
					SessionId: int16(s.smCtx.pduSessionId),
					Transfer:  msg.n2SmInfo,
					N1Sm:      msg.n1Sm,
				}
				if rsp, err := n1n2dl.SessionResourceSetup(s.ranCli, s.ranUeId.Id, req); err == nil {
					//handle response
					if rsp.Success {
						s.smCtx.handleN2SmInfo(models.N2SMINFOTYPE_PDU_RES_SETUP_RSP, rsp.Transfer)
					} else {
						s.smCtx.handleN2SmInfo(models.N2SMINFOTYPE_PDU_RES_SETUP_FAIL, rsp.Transfer)
					}
				} else {
					return utils.WrapError("Request SessionResourceSetup", err)
				}
			}
		}

	} else {
		if s.pending != nil { //finalize this pending
			s.smCtx.Infof("Send N1N2 as UpdateSmContextResponse")
			// build content for UpdateSmContext response
			rsp := &models.UpdateSmContextResponse{
				JsonData: new(models.SmContextUpdatedData),
			}
			rsp.JsonData.UpCnxState = s.smCtx.getUpCnxState()
			if len(msg.n1Sm) > 0 {
				rsp.BinaryDataN1SmMessage = msg.n1Sm
			}
			if len(msg.n2SmInfoType) > 0 {
				rsp.BinaryDataN2SmInformation = msg.n2SmInfo
				rsp.JsonData.N2SmInfoType = msg.n2SmInfoType
			}
			s.pending.Response = rsp
			s.pending.Finalize(nil)
		} else {
			s.smCtx.Infof("Send N1N2 with N12N2MessageTransfer")
			//send N1N2 using N1N2MessageTransferRequest
			req := &models.N1N2MessageTransferRequest{
				BinaryDataN2Information: msg.n2SmInfo,
				BinaryDataN1Message:     msg.n1Sm,
				JsonData: &models.N1N2MessageTransferReqData{
					PduSessionId: new(int),
					TargetAccess: s.smCtx.access,
				},
			}
			*req.JsonData.PduSessionId = int(s.smCtx.pduSessionId)
			if len(msg.n2SmInfo) > 0 {
				req.JsonData.N2InfoContainer = &models.N2InfoContainer{
					N2InformationClass: models.N2INFORMATIONCLASS_SM,
					SmInfo: &models.N2SmInformation{
						PduSessionId: int(s.smCtx.pduSessionId),
						N2InfoContent: &models.N2InfoContent{
							NgapIeType: getNgapIeType(msg.n2SmInfoType),
							NgapData: models.RefToBinaryData{
								ContentId: "N2SmInformation",
							},
						},
						//SNssai: *smCtx.snssai,
					},
				}
			}
			if len(msg.n1Sm) > 0 {
				req.JsonData.N1MessageContainer = &models.N1MessageContainer{
					N1MessageClass: models.N1MESSAGECLASS_SM,
				}
			}
			//TODO: add identity and notification URI for the request
			newId := s.idGen
			if rsp, _, err := comm.N1N2MessageTransfer(s.smCtx.amfCli, s.smCtx.supi, req); rsp != nil {
				//TODO: handle causes
				switch rsp.Cause {
				case models.N1N2MESSAGETRANSFERCAUSE_N1_N2_TRANSFER_INITIATED:
					return nil
				case models.N1N2MESSAGETRANSFERCAUSE_ATTEMPTING_TO_REACH_UE:
					fallthrough
				case models.N1N2MESSAGETRANSFERCAUSE_WAITING_FOR_ASYNCHRONOUS_TRANSFER:
					t := time.NewTimer(N1N2_TIMEOUT)
					pending := &N1N2Pending{
						doneCh: make(chan error, 1), //wait for notification
						id:     newId,
					}
					s.idGen++ //increase the counter of the n1n2 id generator
					s.mu.Lock()
					s.n1n2List[newId] = pending
					s.mu.Unlock()
					select {
					case <-t.C:
						return fmt.Errorf("No response from AMF")
					case err := <-pending.doneCh:
						if err != nil {
							return fmt.Errorf("Receive failure notification for N1N2MessageTransferRequest: %+v", err)
						}
					}

				//case models.N1N2MESSAGETRANSFERCAUSE_UE_NOT_RESPONDING:
				//case models.N1N2MESSAGETRANSFERCAUSE_N1_MSG_NOT_TRANSFERRED:
				//case models.N1N2MESSAGETRANSFERCAUSE_N2_MSG_NOT_TRANSFERRED:
				//case models.N1N2MESSAGETRANSFERCAUSE_UE_NOT_REACHABLE_FOR_SESSION:
				//case models.N1N2MESSAGETRANSFERCAUSE_TEMPORARY_REJECT_REGISTRATION_ONGOING:
				//case models.N1N2MESSAGETRANSFERCAUSE_TEMPORARY_REJECT_HANDOVER_ONGOING:
				//case models.N1N2MESSAGETRANSFERCAUSE_REJECTION_DUE_TO_PAGING_RESTRICTION:
				//case models.N1N2MESSAGETRANSFERCAUSE_AN_NOT_RESPONDING:
				//case models.N1N2MESSAGETRANSFERCAUSE_FAILURE_CAUSE_UNSPECIFIED:
				default:
					return fmt.Errorf("N1N2MessageTransfer not handled at AMF")
				}
			} else {
				if err != nil {
					return utils.WrapError("Request N1N2MessageTransfer", err)
				}
				return fmt.Errorf("Amf response error for N1N2MessageTransfer request")
			}
		}
	}
	return nil
}

type N1N2Messages struct {
	n1Sm         []byte
	n2SmInfo     []byte
	n2SmInfoType models.N2SmInfoType
	/*
		NGAPIETYPE_PDU_RES_SETUP_REQ
		NGAPIETYPE_PDU_RES_REL_CMD
		NGAPIETYPE_PDU_RES_MOD_REQ
		NGAPIETYPE_HANDOVER_CMD
		NGAPIETYPE_HANDOVER_REQUIRED
		NGAPIETYPE_HANDOVER_PREP_FAIL
	*/

}

func getNgapIeType(n2SmInfoType models.N2SmInfoType) (ngapIeType models.NgapIeType) {
	ngapIeType = ""
	switch n2SmInfoType {
	case models.N2SMINFOTYPE_PDU_RES_SETUP_REQ:
		ngapIeType = models.NGAPIETYPE_PDU_RES_SETUP_REQ
	case models.N2SMINFOTYPE_PDU_RES_REL_CMD:
		ngapIeType = models.NGAPIETYPE_PDU_RES_REL_CMD
	case models.N2SMINFOTYPE_PDU_RES_MOD_REQ:
		ngapIeType = models.NGAPIETYPE_PDU_RES_MOD_REQ
	case models.N2SMINFOTYPE_HANDOVER_REQUIRED:
		ngapIeType = models.NGAPIETYPE_HANDOVER_REQUIRED
	case models.N2SMINFOTYPE_HANDOVER_CMD:
		ngapIeType = models.NGAPIETYPE_HANDOVER_CMD
	case models.N2SMINFOTYPE_HANDOVER_PREP_FAIL:
		ngapIeType = models.NGAPIETYPE_HANDOVER_PREP_FAIL
	}
	return
}
