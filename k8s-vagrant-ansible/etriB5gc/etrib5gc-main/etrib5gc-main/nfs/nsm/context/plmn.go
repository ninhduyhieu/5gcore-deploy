package context

import (
	"etrib5gc/nfs/nsm/config"
	"github.com/reogac/sbi/models"
)

type SliceMapping struct {
	serving models.Snssai
	isLbo   bool
}
type PlmnPeer struct {
	id     models.PlmnId                  //home PlmnId
	slices map[models.Snssai]SliceMapping //key = home Snssai, value = serving Snssai
	sepps  map[string]bool
}

func (p *PlmnPeer) toConfig() (info models.HomePlmnConfiguration) {
	info.Id = p.id
	for home, m := range p.slices {
		info.Slices = append(info.Slices, models.MappingOfSnssai{
			HomeSnssai:    home,
			ServingSnssai: m.serving,
			IsLbo:         m.isLbo,
		})
	}
	for sepp, _ := range p.sepps {
		info.Sepps = append(info.Sepps, sepp)
	}
	return
}

func (ctx *NsmContext) createPlmn(info *config.PlmnConfig) (plmn PlmnPeer) {
	plmn.id = info.PlmnId
	plmn.slices = make(map[models.Snssai]SliceMapping)
	for _, s := range info.Slices {
		if ctx.hasSlice(&s.Serving) {
			plmn.slices[s.Home] = SliceMapping{
				serving: s.Serving,
				isLbo:   s.IsLbo,
			}
		}
	}
	plmn.sepps = make(map[string]bool)
	for _, addr := range info.Sepps {
		if len(addr) > 0 {
			plmn.sepps[addr] = true
		}
	}
	return
}
