package ue

import (
	"etrib5gc/nfs/damf/context"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
)

type RequestedSnssai struct {
	value  models.Snssai
	mapped *models.Snssai
}

func buildRequestedSlices(nasSlices *nas.Nssai) (slices []RequestedSnssai) {
	if nasSlices == nil {
		return
	}

	for _, nasSlice := range nasSlices.List {
		slice := RequestedSnssai{
			value: models.Snssai{
				Sst: int(nasSlice.Sst),
				Sd:  nasSlice.GetSd(),
			},
		}
		if nasSlice.Mapped != nil {
			slice.mapped = &models.Snssai{
				Sst: int(nasSlice.Mapped.Sst),
				Sd:  nasSlice.GetMappedSd(),
			}
		}
		slices = append(slices, slice)
	}
	return
}

func isGutiSupported(guti *nas.Guti) bool {
	plmnId := context.PlmnId()
	mcc, mnc := guti.PlmnId.Get()
	return plmnId.Mcc == mcc && plmnId.Mnc == mnc
}
