package secmode

import (
	"etrib5gc/internal/nasbuilder"
	"github.com/reogac/nas"
	"github.com/reogac/utils"
)

func (proc *SecmodeProcedure) buildSecmodeCommand(msg *nas.SecurityModeCommand) error {
	nasCtx := proc.secCtx.NasContext()
	//write SecmodeCommand
	encAlg, intAlg := nasCtx.SelectedAlgorithms() //no panic

	msg.Ngksi = *proc.secCtx.NgKsi()
	msg.SelectedNasSecurityAlgorithms = nas.NewSecurityAlgorithms(intAlg, encAlg)
	msg.SetSecurityHeader(nas.NasSecIntegrityNew)
	return nil

}

func (proc *SecmodeProcedure) sendSecmodeCommand() {
	nasCtx := proc.secCtx.NasContext()
	//write SecmodeCommand
	pdu, err := nasbuilder.NewGmmComposer(nasCtx, proc.isGpp, new(nas.SecurityModeCommand)).Build(proc.buildSecmodeCommand, proc.builder)

	if err != nil {
		proc.callback(nil, utils.WrapError("Build Security Mode Command", err))
	} else if err = proc.sendNasDl(pdu); err != nil {
		proc.callback(nil, utils.WrapError("Send Security Mode Command", err))
	} else {
		proc.Infof("SecurityModeEstablishmentRequest was sent to UE")
		proc.t3560.Start() //start only if sending is success
		proc.t3560Cnt++
	}
}
