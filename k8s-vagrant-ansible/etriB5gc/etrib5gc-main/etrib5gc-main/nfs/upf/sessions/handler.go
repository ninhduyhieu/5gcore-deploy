package sessions

import (
	"etrib5gc/nfs/upf/report"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/pfcp/message"
	"github.com/reogac/utils"
	"time"
)

func (s *PfcpSession) HandleSessionEstablishment(req *message.PFCPSessionEstablishmentRequest) (rsp *message.PFCPSessionEstablishmentResponse, err error) {
	s.Infoln("Handle SessionEstablishmentRequest")
	tasks := []func(chan error){}
	for _, i := range req.CreateFAR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createFAR(&i); err != nil {
				errCh <- utils.WrapError("Est CreateFAR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.CreateQER {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createQER(&i); err != nil {
				errCh <- utils.WrapError("Est CreateQER", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.CreateURR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createURR(&i); err != nil {
				errCh <- utils.WrapError("Est CreateURR", err)
			} else {
				errCh <- nil
			}
		})
	}

	if req.CreateBAR != nil {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createBAR(req.CreateBAR); err != nil {
				errCh <- utils.WrapError("Est CreateBAR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.CreatePDR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createPDR(&i); err != nil {
				errCh <- utils.WrapError("Est CreatePDR", err)
			} else {
				errCh <- nil
			}
		})
	}

	rsp = &message.PFCPSessionEstablishmentResponse{
		NodeID:  _sessMan.nodeId,
		UPFSEID: &ie.FSEID{},
	}
	if newErr := doTasks(tasks); newErr != nil {
		s.Errorf("Fail to send rules to GTP kernel: %+v", newErr)
		rsp.Cause = ie.CauseRequestRejected
	} else {
		rsp.Cause = ie.CauseRequestAccepted
	}
	if s.fSEID() != nil {
		rsp.UPFSEID = s.fSEID()
	}
	s.Infoln("Created NewSessionEstablishmentResponse")

	return
}

func (s *PfcpSession) HandleSessionModification(req *message.PFCPSessionModificationRequest) (rsp *message.PFCPSessionModificationResponse, err error) {
	s.Infoln("Handle SessionModificationRequest")

	tasks := []func(chan error){}

	for _, i := range req.CreateFAR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createFAR(&i); err != nil {
				errCh <- utils.WrapError("Mod createFAR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.CreateQER {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createQER(&i); err != nil {
				errCh <- utils.WrapError("Mod createQER", err)
			} else {
				errCh <- nil
			}
		})

	}

	for _, i := range req.CreateURR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createURR(&i); err != nil {
				errCh <- utils.WrapError("Mod createURR", err)
			} else {
				errCh <- nil
			}
		})
	}

	if req.CreateBAR != nil {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createBAR(req.CreateBAR); err != nil {
				errCh <- utils.WrapError("Mod createBAR", err)
			} else {
				errCh <- nil
			}
		})

	}

	for _, i := range req.CreatePDR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.createPDR(&i); err != nil {
				errCh <- utils.WrapError("Mod createPDR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.RemoveFAR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.removeFAR(&i); err != nil {
				errCh <- utils.WrapError("Mod removeFAR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.RemoveQER {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.removeQER(&i); err != nil {
				errCh <- utils.WrapError("Mod removeQER", err)
			} else {
				errCh <- nil
			}
		})
	}

	var usars []report.USAReport

	for _, i := range req.RemoveURR {
		tasks = append(tasks, func(errCh chan error) {
			if rs, err := s.removeURR(&i); err != nil {
				errCh <- utils.WrapError("Mod removeURR", err)
			} else {
				if len(rs) > 0 {
					usars = append(usars, rs...)
				}
				errCh <- nil
			}
		})
	}

	if req.RemoveBAR != nil {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.removeBAR(req.RemoveBAR); err != nil {
				errCh <- utils.WrapError("Mod removeBAR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.RemovePDR {
		tasks = append(tasks, func(errCh chan error) {
			if rs, err := s.removePDR(&i); err != nil {
				errCh <- utils.WrapError("Mod removePDR", err)
			} else {
				if len(rs) > 0 {
					usars = append(usars, rs...)
				}
				errCh <- nil
			}
		})
	}

	for _, i := range req.UpdateFAR {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.updateFAR(&i); err != nil {
				errCh <- utils.WrapError("Mod updateFAR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.UpdateQER {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.updateQER(&i); err != nil {
				errCh <- utils.WrapError("Mod updateQER", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.UpdateURR {
		tasks = append(tasks, func(errCh chan error) {
			if rs, err := s.updateURR(&i); err != nil {
				errCh <- utils.WrapError("Mod updateURR", err)
			} else {
				if len(rs) > 0 {
					usars = append(usars, rs...)
				}
				errCh <- nil
			}
		})
	}

	if req.UpdateBAR != nil {
		tasks = append(tasks, func(errCh chan error) {
			if err := s.updateBAR(req.UpdateBAR); err != nil {
				errCh <- utils.WrapError("Mod updateBAR", err)
			} else {
				errCh <- nil
			}
		})
	}

	for _, i := range req.UpdatePDR {
		tasks = append(tasks, func(errCh chan error) {
			if rs, err := s.updatePDR(&i); err != nil {
				errCh <- utils.WrapError("Mod updatePDR", err)
			} else {
				if len(rs) > 0 {
					usars = append(usars, rs...)
				}
				errCh <- nil
			}
		})
	}

	if req.QueryURR != nil {
		tasks = append(tasks, func(errCh chan error) {
			if qUrr, err := s.queryURR(req.QueryURR); err != nil {
				errCh <- utils.WrapError("Mod queryURR", err)
			} else {
				if len(qUrr) > 0 {
					usars = append(usars, qUrr...)
				}
				errCh <- nil
			}
		})
	}
	rsp = &message.PFCPSessionModificationResponse{}

	if newErr := doTasks(tasks); newErr != nil {
		s.Errorf("Fail to send rules to GTP kernel: %+v", newErr)
		rsp.Cause = ie.CauseSystemFailure
	} else {
		rsp.Cause = ie.CauseRequestAccepted
	}

	if len(usars) > 0 {
		for _, r := range usars {
			urrInfo, ok := s.urrIdList[r.URRID]
			if !ok {
				s.Errorf("Sess Mod: URRInfo[%#x] not found", r.URRID)
				continue
			}
			r.URSEQN = s.URRSeq(r.URRID)

			if urrInfo.removed {
				delete(s.urrIdList, r.URRID) //////////////
			}
		}
	}

	s.Infoln("Created NewSessionModificationResponse")
	return
}

func (s *PfcpSession) HandleSessionDeletion(req *message.PFCPSessionDeletionRequest) (rsp *message.PFCPSessionDeletionResponse, err error) {
	s.Infoln("Handle SessionDeletionRequest")

	rsp = &message.PFCPSessionDeletionResponse{
		Cause: ie.CauseRequestAccepted,
	}
	return
	var usars []report.USAReport

	usars = s.close()

	for _, r := range usars {
		urrInfo, ok := s.urrIdList[r.URRID]
		if !ok {
			s.Errorf("Sess Del: URRInfo[%#x] not found", r.URRID)
			continue
		}
		r.URSEQN = s.URRSeq(r.URRID)
		// indicates usage report being reported for a URR due to the termination of the PFCP session
		r.USARTrigger.Flags |= report.USAR_TRIG_TERMR ///////////

		if urrInfo.removed {
			delete(s.urrIdList, r.URRID)
		}
	}

	return
}

func doTasks(tasks []func(errCh chan error)) error {
	log.Infof("Sending %d requests to GTP kernel", len(tasks))
	start := time.Now()
	/*
		//TODO: fix paralel sending rule updates to kernel
			errCh := make(chan error, len(tasks))
			if len(tasks) > 1 {
				for _, t := range tasks {
					go t(errCh)
				}
			} else if len(tasks) == 0 {
				return
			} else {
				tasks[0](errCh)
			}

			for i := 0; i < len(tasks); i++ {
				if err = <-errCh; err != nil {
					return
				}
			}
	*/
	errCh := make(chan error, 1)
	for _, t := range tasks {
		t(errCh)
		if err := <-errCh; err != nil {
			return utils.WrapError("Send rule to kernel", err)
		}
	}

	log.Infof("All tasks accompleted in %d milliseconds!", time.Since(start).Milliseconds())
	return nil
}
