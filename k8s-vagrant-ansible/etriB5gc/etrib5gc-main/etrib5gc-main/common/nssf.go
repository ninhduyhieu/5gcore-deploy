package common

import (
	"encoding/hex"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"strings"
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

func getSubscribedSlices(nssai *models.Nssai) (subscribedSlices map[models.Snssai]bool) {
	if nssai == nil {
		return
	}
	subscribedSlices = make(map[models.Snssai]bool)
	for _, snssai := range nssai.DefaultSingleNssais {
		subscribedSlices[snssai] = true
	}
	for _, snssai := range nssai.SingleNssais {

		if _, ok := subscribedSlices[snssai]; !ok {
			subscribedSlices[snssai] = false
		}
	}
	return
}

func GetAllowedSlices(isRoaming bool, nasRequestedSlices *nas.Nssai, ueSlices *models.Nssai, sliceMapping map[models.Snssai]models.Snssai) (allowedSlices []models.AllowedSnssai, err error) {
	subscribedSlices := getSubscribedSlices(ueSlices)
	if len(subscribedSlices) == 0 {
		//TODO: set configured slice
		err = fmt.Errorf("No subsribed slice")
		return
	}
	var requestedSlices []RequestedSnssai
	if nasRequestedSlices != nil {
		requestedSlices = buildRequestedSlices(nasRequestedSlices)
	}
	if len(requestedSlices) == 0 { //UE does not send requested slices
		if !isRoaming { //not roaming, add all subscribed slices
			for slice, _ := range subscribedSlices {
				allowedSlices = append(allowedSlices, models.AllowedSnssai{
					AllowedSnssai: slice,
				})
			}
		} else { //roaming, add all mapped subscribed slices
			for slice, _ := range subscribedSlices {
				if servingSlice, ok := sliceMapping[slice]; ok {
					allowedSlices = append(allowedSlices, models.AllowedSnssai{
						AllowedSnssai:    servingSlice,
						MappedHomeSnssai: &slice,
					})
				}
			}
		}
		return
	}

	for _, slice := range requestedSlices {
		if !isRoaming { //not roaming, requested slice is from serving network
			if _, ok := subscribedSlices[slice.value]; ok {
				allowedSlices = append(allowedSlices, models.AllowedSnssai{
					AllowedSnssai: slice.value,
				})
			}
		} else {
			//get home network slice from requested slice
			homeSlice := &slice.value
			if slice.mapped != nil {
				homeSlice = slice.mapped
			}

			if _, ok := subscribedSlices[*homeSlice]; ok {
				if servingSlice, ok := sliceMapping[*homeSlice]; ok {
					allowedSlices = append(allowedSlices, models.AllowedSnssai{
						AllowedSnssai:    servingSlice,
						MappedHomeSnssai: homeSlice,
					})
				}
			}
		}
	}

	return
}

//validate a slice format
func RewriteSlice(s *models.Snssai) (*models.Snssai, error) {
	if len(s.Sd) == 0 {
		return s, nil
	}
	if v, err := hex.DecodeString(strings.TrimPrefix(s.Sd, "0x")); err == nil && len(v) == 3 {
		return &models.Snssai{
			Sd:  fmt.Sprintf("%x", v),
			Sst: s.Sst,
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("Fail to decode Sd: %+v", err)
	} else {
		return nil, fmt.Errorf("Non-emppty Sd must be 6 hex digits")
	}
}
