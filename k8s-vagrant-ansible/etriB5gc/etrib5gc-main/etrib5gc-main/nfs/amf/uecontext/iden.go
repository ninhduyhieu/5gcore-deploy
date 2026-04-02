package uecontext

import (
	"etrib5gc/nfs/amf/procs/iden"
	"github.com/reogac/nas"
)

// start an identification procedure
func (ueCtx *UeContext) identifyUe() {
	attCtx := ueCtx.attCtx
	ueCtx.Infof("Start identification procedure")
	idType := nas.MobileIdentity5GSTypeSuci
	attCtx.proc = iden.Start(&iden.Context{
		Log:      ueCtx.Entry,
		Callback: ueCtx.onIdentificationComplete,
		IdType:   idType,
		SendNasDl: func(pdu []byte) error {
			return ueCtx.sendNas(pdu, attCtx.isGpp)
		},
		IsGpp:   attCtx.isGpp,
		NasCtx:  ueCtx.getNasContext(),
		OnT3570: ueCtx.onT3570,
	})
}

func (ueCtx *UeContext) onIdentificationComplete(id *nas.MobileIdentity, err error) {
	//detached identification procedure
	ueCtx.attCtx.proc = nil

	if err != nil {
		ueCtx.Errorf("Identification fails: %+v", err)
		n1Cause := nas.Cause5GMMUEIdentityCannotBeDerivedByTheNetwork
		ueCtx.abortAttachmentProcedure(&n1Cause)
	} else if err = ueCtx.updateId(id); err != nil {
		ueCtx.Errorf("Update Ue identity fail: %+v", err)
		n1Cause := nas.Cause5GMMUEIdentityCannotBeDerivedByTheNetwork
		ueCtx.abortAttachmentProcedure(&n1Cause)
	} else {
		ueCtx.goNextRegistrationStep()
	}
}
