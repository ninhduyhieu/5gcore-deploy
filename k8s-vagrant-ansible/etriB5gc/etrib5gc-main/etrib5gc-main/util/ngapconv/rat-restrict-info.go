package ngapconv

import (
	"github.com/reogac/sbi/models"
)

// TS 38.413 9.3.1.85
func RATRestrictionInformationToNgap(ratType models.RatType) (ratResInfo []byte) {
	// Only support EUTRA & NR in version15.2.0
	switch ratType {
	case models.RATTYPE_EUTRA:
		ratResInfo = []byte{0x80}
	case models.RATTYPE_NR:
		ratResInfo = []byte{0x40}
	}
	return
}
