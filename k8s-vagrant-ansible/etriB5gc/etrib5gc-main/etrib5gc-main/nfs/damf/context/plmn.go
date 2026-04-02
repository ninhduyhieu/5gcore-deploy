package context

import (
	"github.com/reogac/sbi/models"
)

type PlmnPeer struct {
	id     models.PlmnId                   //home PlmnId
	slices map[models.Snssai]models.Snssai //key = home Snssai, value = serving Snssai
}
