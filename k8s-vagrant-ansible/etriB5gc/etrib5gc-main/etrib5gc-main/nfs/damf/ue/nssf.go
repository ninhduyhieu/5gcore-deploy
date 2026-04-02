package ue

import (
	"etrib5gc/common"
	"etrib5gc/mesh"
	"etrib5gc/nfs/damf/context"
	"etrib5gc/nfs/damf/nssf"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"strings"
)

func (ueCtx *UeContext) findAmf() (err error) {
	var allowedSlices []models.Snssai

	ueCtx.Trace("Determine allowed slices for UE")
	homePlmnId := ueCtx.plmnId()
	isRoaming := !common.IsPlmnIdEqual(homePlmnId, context.PlmnId())
	if list, err := common.GetAllowedSlices(isRoaming, ueCtx.requestedSlices, ueCtx.amData.Nssai, nssf.GetSliceMapping(homePlmnId)); err != nil {
		return utils.WrapError("Get allowd slices", err)
	} else {
		for _, s := range list {
			allowedSlices = append(allowedSlices, s.AllowedSnssai)
		}
	}

	if len(allowedSlices) == 0 {
		return fmt.Errorf("No allowed slice found")
	}

	allowedSliceList := []string{}
	for _, slice := range allowedSlices {
		allowedSliceList = append(allowedSliceList, slice.String())
	}
	allowedSliceListStr := strings.Join(allowedSliceList, ", ")
	ueCtx.Tracef("Alowed Snssai list = %s", allowedSliceListStr)

	amfSetList := nssf.FindAmf(allowedSlices)
	found := false
	for len(amfSetList) > 0 {
		amfService := common.AmfServiceName(context.PlmnId(), amfSetList[0])
		if ueCtx.amfCli, err = mesh.Consumer(amfService, ueCtx.metadata); err != nil {
			ueCtx.Warnf("Fail to create AMF consumer with AmfSet=%s: %+v", amfSetList[0], err)
			amfSetList = amfSetList[1:]
		} else {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Fail to select an AMF for slice(s): %s", allowedSliceListStr)
	}

	return
}
