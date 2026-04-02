package common

import (
	"fmt"
	"github.com/reogac/sbi/models"
)

func UpfServiceName(plmnId *models.PlmnId) string {
	return fmt.Sprintf("upf-%s", models.PlmnIdToString(*plmnId))
}

func NsmServiceName(plmnId *models.PlmnId) string {
	return fmt.Sprintf("nsm-%s", models.PlmnIdToString(*plmnId))
}

func DamfServiceName(plmnId *models.PlmnId) string {
	return fmt.Sprintf("damf-%s", models.PlmnIdToString(*plmnId))
}

func AmfServiceName(plmnId *models.PlmnId, amfSet string) string {
	if len(amfSet) == 0 {
		return fmt.Sprintf("amf-%s", models.PlmnIdToString(*plmnId))
	} else {
		return fmt.Sprintf("amf-%s-%s", models.PlmnIdToString(*plmnId), amfSet)
	}
}

func SmfServiceName(plmnId *models.PlmnId, slice *models.Snssai) string {
	if slice != nil {
		return fmt.Sprintf("smf-%s-%s", models.PlmnIdToString(*plmnId), slice.String())
	} else {
		return fmt.Sprintf("smf-%s", models.PlmnIdToString(*plmnId))
	}
}

func UdmServiceName(plmnId *models.PlmnId) string {
	return fmt.Sprintf("udm-%s", models.PlmnIdToString(*plmnId))
}

func PcfServiceName(plmnId *models.PlmnId) string {
	return fmt.Sprintf("pcf-%s", models.PlmnIdToString(*plmnId))
}

func AusfServiceName(plmnId *models.PlmnId) string {
	return fmt.Sprintf("ausf-%s", models.PlmnIdToString(*plmnId))
}

func RanServiceName(plmnId *models.PlmnId, id string) string {
	return fmt.Sprintf("pran-%s-%s", models.PlmnIdToString(*plmnId), id)
}
