package uecontext

import (
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"strings"
	"sync"
)

func isGpp(access models.AccessType) bool {
	return access == models.ACCESSTYPE_3GPP_ACCESS
}

func getAccessType(isGpp bool) models.AccessType {
	if isGpp {
		return models.ACCESSTYPE_3GPP_ACCESS
	}
	return models.ACCESSTYPE_NON_3GPP_ACCESS
}

func buildNasAllowedSnssai(slices []models.AllowedSnssai) *nas.Nssai {
	log.Infof("Build NAS Allowed Nssai: %+v", slices)
	nasSlices := new(nas.Nssai)
	for _, slice := range slices {
		nasSlice := new(nas.SNssai)
		nasSlice.Set(uint8(slice.AllowedSnssai.Sst), slice.AllowedSnssai.Sd)
		if slice.MappedHomeSnssai != nil {
			nasSlice.SetMapped(uint8(slice.MappedHomeSnssai.Sst), slice.MappedHomeSnssai.Sd)
		} else {
			nasSlice.SetMapped(uint8(slice.AllowedSnssai.Sst), slice.AllowedSnssai.Sd)
		}
		nasSlices.Add(nasSlice)
	}
	return nasSlices
}

func buildServiceAreaList(areas []models.Tai) *nas.TrackingAreaIdentityList {
	taiList := new(nas.TaiListType2) //always use type 2 - no optimization
	for _, tai := range areas {
		if nasTai := tai.NasType(); nasTai != nil {
			taiList.List = append(taiList.List, *nasTai)
		}
	}
	if len(taiList.List) > 0 {
		return &nas.TrackingAreaIdentityList{
			Lists: []nas.TaiListInf{
				taiList,
			},
		}
	}
	return nil
}

// convert boolean value to uint8 (0 or 1)
func b2uint8(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}
func newUint8(v uint8) *uint8 {
	return &v
}

func getN1Type(n1MsgClass models.N1MessageClass) (n1Type uint8) {
	switch n1MsgClass {
	case models.N1MESSAGECLASS_SM:
		n1Type = nas.PayloadContainerTypeN1SMInfo
	case models.N1MESSAGECLASS_SMS:
		n1Type = nas.PayloadContainerTypeSMS
	case models.N1MESSAGECLASS_LPP:
		n1Type = nas.PayloadContainerTypeLPP
	case models.N1MESSAGECLASS_UPDP:
		n1Type = nas.PayloadContainerTypeUEPolicy
	default:
		//n1Type = 0 //unknown
		n1Type = nas.PayloadContainerTypeN1SMInfo //default to be N1Sm
	}
	return
}
func getN2SmInfoType(ieType models.NgapIeType) (n2SmInfoType models.N2SmInfoType) {
	switch ieType {
	case models.NGAPIETYPE_PDU_RES_SETUP_REQ:
		n2SmInfoType = models.N2SMINFOTYPE_PDU_RES_SETUP_REQ

	case models.NGAPIETYPE_PDU_RES_MOD_REQ:
		n2SmInfoType = models.N2SMINFOTYPE_PDU_RES_MOD_REQ

	case models.NGAPIETYPE_PDU_RES_REL_CMD:
		n2SmInfoType = models.N2SMINFOTYPE_PDU_RES_REL_CMD

	default:
		n2SmInfoType = ""
	}
	return
}

func extractAmPolId(headers map[string]string) string {
	return extractIdFromLocation(headers)
}

func extractIdFromLocation(headers map[string]string) string {
	if v, ok := headers["Location"]; ok {
		if parts := strings.Split(v, "/"); len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

//send tasks to worker pool and wait for their completion
func executeTasks(tasks []func()) {
	numTasks := len(tasks)
	if numTasks == 0 {
		return
	}
	if numTasks == 1 {
		tasks[0]()
	} else {
		wg := new(sync.WaitGroup)
		wg.Add(numTasks)
		for _, task := range tasks {
			_uePool.pubWorkers.Go(func() {
				task()
				wg.Done()
			})
			//go task()
		}
		wg.Wait()
	}
}

//send tasks to worker pool
func sendTasks(tasks []func()) {
	numTasks := len(tasks)
	if numTasks == 0 {
		return
	}
	for _, task := range tasks {
		_uePool.pubWorkers.Go(task)
	}
}
