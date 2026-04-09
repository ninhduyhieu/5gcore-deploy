package context

import (
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/idgen"
	"strings"
)

const (
	MAX_6_BITS uint8 = 1<<7 - 1
)

// AmfSet = AMFRegion (8bit)+AMFSet(10bit)
// AMFPointer(6bit) is generated from the set
type AmfSet struct {
	id         string //18bit
	pointerGen idgen.Generator[uint8]
	amfList    map[string]*Amf
	slices     map[string]models.Snssai
}

// check if the region include the amf set
func (s *AmfSet) inRegion(region string) bool {
	parts := strings.Split(s.id, "-")
	return parts[0] == region
}

func (s *AmfSet) getSlices() (slices []models.Snssai) {
	for _, s := range s.slices {
		slices = append(slices, s)
	}
	return
}
func (s *AmfSet) removeAmf(uuid string) {
	log.Warnf("AMF %s removed", uuid)
	if amf, ok := s.amfList[uuid]; ok {
		s.pointerGen.Free(amf.pointer)
		delete(s.amfList, uuid)
	}
}

func (s *AmfSet) createConfigModel() (cfg models.AmfSetConfiguration) {
	cfg.SetId = s.id
	cfg.Slices = s.getSlices()
	return
}

type Amf struct {
	uuid    string
	setId   string
	pointer uint8
}
