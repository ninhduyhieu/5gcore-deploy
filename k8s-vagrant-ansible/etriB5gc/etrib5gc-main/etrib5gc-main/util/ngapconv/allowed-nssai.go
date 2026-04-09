package ngapconv

import (
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
)

func AllowedNssaiToNgap(allowedNssaiModels []models.AllowedSnssai) (items []ies.AllowedNSSAIItem, err error) {
	var snssai *ies.SNSSAI
	for _, allowedSnssai := range allowedNssaiModels {
		item := &ies.AllowedNSSAIItem{}

		if snssai, err = SNssaiToNgap(allowedSnssai.AllowedSnssai); err != nil {
			return
		}
		item.SNSSAI = *snssai
		items = append(items, *item)

	}
	return
}

func AllowedNssaiToModels(items []ies.AllowedNSSAIItem) (allowedNssaiModels []models.AllowedSnssai) {
	for _, item := range items {
		snssai := SNssaiToModels(item.SNSSAI)
		allowedSnssai := models.AllowedSnssai{
			AllowedSnssai: snssai,
		}
		allowedNssaiModels = append(allowedNssaiModels, allowedSnssai)
	}
	return
}
