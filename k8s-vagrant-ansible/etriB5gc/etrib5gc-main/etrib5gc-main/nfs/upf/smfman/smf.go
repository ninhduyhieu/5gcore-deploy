package smfman

import (
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
)

type Smf struct {
	smfCli         sbi.ConsumerClient
	id             string
	slice          models.Snssai
	teidRangeIndex int
}

func (smf *Smf) SbiClient() sbi.ConsumerClient {
	return smf.smfCli
}
