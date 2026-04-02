package nssf

import (
	"etrib5gc/common"
	"etrib5gc/logctx"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
	"strings"
)

type Plmn struct {
	id     models.PlmnId
	slices map[models.Snssai]models.Snssai //map home slice -> serving slice
}

type AmfSet struct {
	id     string
	slices map[string]models.Snssai
}

func (s *AmfSet) canServe(slices []models.Snssai) bool {
	for _, slice := range slices {
		if _, ok := s.slices[slice.String()]; !ok {
			return false
		}
	}
	return true
}

type NSSF struct {
	slices    map[string]models.Snssai //serving slices
	plmnPeers map[models.PlmnId]Plmn
	amfSets   map[string]*AmfSet
}

var _nssf *NSSF
var log *logrus.Entry

func Init(cfg *models.NssfConfiguration) {
	if _nssf != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "nssf",
	})
	_nssf = &NSSF{
		amfSets:   make(map[string]*AmfSet),
		slices:    make(map[string]models.Snssai),
		plmnPeers: make(map[models.PlmnId]Plmn),
	}

	_nssf.loadNssfConfig(cfg)
}

func (m *NSSF) hasSlice(s *models.Snssai) bool {
	_, ok := m.slices[s.String()]
	return ok
}

func (m *NSSF) createPlmn(info *models.HomePlmnConfiguration) (plmn Plmn) {
	plmn.id = info.Id
	plmn.slices = make(map[models.Snssai]models.Snssai)
	for _, s := range info.Slices {
		if m.hasSlice(&s.ServingSnssai) {
			plmn.slices[s.HomeSnssai] = s.ServingSnssai
		}
	}
	return
}

func (m *NSSF) loadNssfConfig(cfg *models.NssfConfiguration) {
	for _, item := range cfg.Slices {
		m.slices[item.Id.String()] = item.Id
	}

	for _, item := range cfg.PlmnPeers {
		m.plmnPeers[item.Id] = m.createPlmn(&item)
	}

	for _, item := range cfg.AmfSets {
		if _, err := common.AmfSetFromString(item.SetId); err != nil { //check format (AmfRegion + AmfSet)
			log.Warnf("AmfSet %s is invalid", item.SetId)
			continue
		}

		amfset := &AmfSet{
			id:     item.SetId,
			slices: make(map[string]models.Snssai),
		}
		for _, slice := range item.Slices {
			amfset.slices[slice.String()] = slice
		}
		m.amfSets[amfset.id] = amfset

		log.Infof("Add AmfSet %s: %v", item.SetId, item.Slices)
	}
}

func FindAmf(slices []models.Snssai) []string {
	list := []string{}
	for _, amfSet := range _nssf.amfSets {
		if amfSet.canServe(slices) {
			list = append(list, amfSet.id)
		}
	}

	log.Debugf("List AmfSet can serve the UE's slices: %s", strings.Join(list, ", "))
	return list
}

func GetSliceMapping(plmnId *models.PlmnId) map[models.Snssai]models.Snssai {
	if plmn, ok := _nssf.plmnPeers[*plmnId]; ok {
		return plmn.slices
	}
	return nil
}
func GetServingSlice(plmnId *models.PlmnId, homeSlice *models.Snssai) *models.Snssai {
	if plmn, ok := _nssf.plmnPeers[*plmnId]; ok {
		if servingSlice, ok := plmn.slices[*homeSlice]; ok {
			return &servingSlice
		}
	}
	return nil
}
