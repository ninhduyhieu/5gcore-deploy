package ngapconv

import (
	"github.com/reogac/sbi/models"
	"strings"
	"testing"
)

func TestAmfId(t *testing.T) {
	amfid := "112233"
	if a, b, c, err := AmfIdToNgap(amfid); err != nil {
		t.Errorf(err.Error())
	} else {
		newamfid := AmfIdToString(a, b, c)
		if strings.Compare(amfid, newamfid) != 0 {
			t.Errorf("%s vs %s", amfid, newamfid)
		}
	}
}

func TestPlmnId(t *testing.T) {
	plmnId := models.PlmnId{
		Mcc: "208",
		Mnc: "93",
	}
	if ngapPlmnId, err := PlmnIdToNgap(plmnId); err != nil {
		t.Errorf(err.Error())
	} else {
		newPlmnId := PlmnIdToModels(ngapPlmnId)
		if strings.Compare(plmnId.String(), newPlmnId.String()) != 0 {
			t.Errorf("Not Match")
		}
	}
}
