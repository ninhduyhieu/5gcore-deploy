package ue

import (
	"github.com/reogac/sbi/models"
)

func (ueCtx *UeContext) selectNewNgKsi(old *models.NgKsi) (ngKsi models.NgKsi) {
	ngKsi.Tsc = models.SCTYPE_NATIVE
	if old == nil || old.Ksi >= 7 {
		ngKsi.Ksi = 0
	} else {
		if ngKsi.Ksi = old.Ksi + 1; ngKsi.Ksi == 7 {
			ngKsi.Ksi = 0
		}
	}
	ueCtx.Debugf("Select new NgKsi for authentication: %s", ngKsi.String())
	return
}
