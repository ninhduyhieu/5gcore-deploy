package uecontext

import (
	"fmt"
	"github.com/reogac/sbi/models"
)

func (ueCtx *UeContext) DoPathSwitch(isGpp bool, req *models.PathSwitchRequest) (*models.PathSwitchAcknowledge, *models.PathSwitchFailure, error) {
	//1. check for register status
	if !ueCtx.isRegistered(isGpp) {
		return nil, nil, fmt.Errorf("Ue not registered")
	}

	//2 check for security context
	if !ueCtx.HasValidSecmode() {
		return nil, nil, fmt.Errorf("No security context")
	}
	//3. update ue security capability
	ueCtx.updateSecurityCapability(req.UeSecurityCapability)

	//4. now tell SMF what to to with the Pdu sessions
	var n2DlList []models.N2SmInfoDownlinkContent
	jobs := []func(){}

	for _, session := range req.Sessions {
		//setup sessions
		//forward downlink acknowledgement/failure N2SmInfo messages
		msg := &models.SmContextUpdateData{
			FailedToBeSwitched: new(bool),
			N2SmInfoType:       session.N2SmInfoType,
		}

		switch session.N2SmInfoType {
		case models.N2SMINFOTYPE_PATH_SWITCH_REQ:
			*msg.FailedToBeSwitched = false
		case models.N2SMINFOTYPE_PATH_SWITCH_SETUP_FAIL:
			*msg.FailedToBeSwitched = true
		default:
			ueCtx.Warnf("Invalid N2SmInfo type for session %d", session.SessionId)
			continue
		}
		if smCtx := ueCtx.findSmContext(uint8(session.SessionId)); smCtx != nil {
			jobs = append(jobs, func() {
				if rsp, ersp, err := smCtx.SendUpdateSmContext(msg, nil, session.N2SmInfo); err != nil {
					ueCtx.Errorf("Fail to forward PathSwitch transfer for session %d", session.SessionId)
				} else {
					if n2, _ := ueCtx.processUpdateSmContextResponses(smCtx, rsp, ersp); n2 != nil {
						n2DlList = append(n2DlList, *n2)
					}
				}
			})
		} else {
			ueCtx.Errorf("Session %d not found", session.SessionId)
		}
	}

	//execute all jobs to request updates at SMFs
	executeTasks(jobs)

	secCtx := models.SecurityContext{
		Ncc: int16(ueCtx.currentSecCtx.Ncc()),
		Nh:  ueCtx.currentSecCtx.Nh(),
	}

	//5. update ran security parameters for the next handover
	ueCtx.currentSecCtx.UpdateNh()

	//6. create response
	return &models.PathSwitchAcknowledge{
		Sessions:             n2DlList,
		UeSecurityCapability: ueCtx.getUeSecurityCapability(),
		AllowedNssai: models.AllowedNssai{
			AllowedSnssaiList: ueCtx.getAllowedSlices(isGpp),
			AccessType:        getAccessType(isGpp),
		},
		SecurityContext: secCtx,
	}, nil, nil
}
