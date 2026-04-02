package uecontext

import (
	"etrib5gc/logctx"
	"github.com/alitto/pond/v2"
	"github.com/reogac/nas"
	"github.com/reogac/utils/idgen"
	"github.com/sirupsen/logrus"
	"math"
	"sync"
)

var _uePool *UePool
var log *logrus.Entry

func InitUePool(pub pond.Pool) {
	if _uePool != nil {
		return
	}

	log = logctx.Entry(logrus.Fields{
		"mod": "uectx",
	})
	_uePool = &UePool{
		pubWorkers: pub,
		tmsiGen:    idgen.NewGenerator[uint32](1, math.MaxInt32),
		n1n2man:    newN1N2Man(),
		byTmsi:     make(map[uint32]*UeContext),
		bySupi:     make(map[string]*UeContext),
		byPei:      make(map[string]*UeContext),
	}
	//initialize state machine
	initFsm()
}

func CloseUePool() {
	if _uePool != nil {
		_uePool.n1n2man.clean()
		_uePool.removeAll()
		_uePool = nil
		_sm.Stop()
	}
}

// UePool has all UeContexts which are indexed with suci, supi, tmsi5gs and pei
type UePool struct {
	pubWorkers pond.Pool //for public jobs
	tmsiGen    idgen.Generator[uint32]
	byTmsi     map[uint32]*UeContext
	bySupi     map[string]*UeContext
	byPei      map[string]*UeContext
	mutex      sync.RWMutex

	n1n2man N1N2Man
}

func FindUeByTmsi(tmsi uint32) (ueCtx *UeContext) {
	return _uePool.findUeByTmsi(tmsi)
}

func FindUeBySupi(supi string) *UeContext {
	return _uePool.findUeBySupi(supi)
}

// try to extract ue identity then find its context
func findUeByMobileId(mobileId *nas.MobileIdentity) *UeContext {
	idType := mobileId.GetType()

	switch idType {
	case nas.MobileIdentity5GSTypeSuci:
		return nil //NOTE: SUCI changes every registration

	case nas.MobileIdentity5GSType5gGuti:
		guti := mobileId.Id.(*nas.Guti)
		log.Infof("Look for UeContext with GUTI-TMSI: %d", guti.Tmsi)
		return _uePool.findUeByTmsi(guti.Tmsi)

	case nas.MobileIdentity5GSType5gSTmsi:
		//TODO: check if AmfId matched
		tmsi5gs := mobileId.Id.(*nas.Tmsi5Gs)
		log.Infof("Look for UeContext with 5G-S-TMSI: %d", tmsi5gs.Tmsi)
		return _uePool.findUeByTmsi(tmsi5gs.Tmsi)

	case nas.MobileIdentity5GSTypeImei, nas.MobileIdentity5GSTypeImeisv:
		return _uePool.findUeByPei(mobileId.String())
	default:
		//do nothing
	}
	return nil
}

func (p *UePool) findUeBySupi(supi string) *UeContext {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ueCtx, _ := p.bySupi[supi]
	return ueCtx
}

func (p *UePool) findUeByTmsi(tmsi uint32) *UeContext {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ueCtx, _ := p.byTmsi[tmsi]
	return ueCtx
}

func (p *UePool) findUeByPei(pei string) *UeContext {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	ueCtx, _ := p.byPei[pei]
	return ueCtx
}

func (p *UePool) add(ueCtx *UeContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	//allocate TMSI (so GUTI and TMSI5GS)
	ueCtx.tmsi = p.tmsiGen.Allocate()
	log.Infof("New UeContext is create [tmsi=%d]", ueCtx.tmsi)

	p.byTmsi[ueCtx.tmsi] = ueCtx
	if len(ueCtx.supi) > 0 {
		p.bySupi[ueCtx.supi] = ueCtx
	}
	if len(ueCtx.pei) > 0 {
		p.byPei[ueCtx.pei] = ueCtx
	}
}

func (p *UePool) removeAll() {
	p.mutex.Lock()

	ueList := []*UeContext{}

	for _, ueCtx := range p.byTmsi {
		ueList = append(ueList, ueCtx)
	}
	p.mutex.Unlock()

	var wg sync.WaitGroup
	for _, ueCtx := range ueList {
		wg.Add(1)
		go func(ue *UeContext, wg *sync.WaitGroup) {
			ue.kill()
			wg.Done()
		}(ueCtx, &wg)
	}
	wg.Wait()
	log.Infof("%d UeContext(s) removed", len(ueList))

	p.bySupi = make(map[string]*UeContext)
	p.byPei = make(map[string]*UeContext)
	p.byTmsi = make(map[uint32]*UeContext)
}

func (p *UePool) updateUePei(pei string, ueCtx *UeContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	_uePool.byPei[pei] = ueCtx
}

func (p *UePool) updateUeSupi(supi string, ueCtx *UeContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	_uePool.bySupi[supi] = ueCtx
}

func (p *UePool) remove(ueCtx *UeContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.tmsiGen.Free(ueCtx.tmsi)
	delete(p.bySupi, ueCtx.supi)
	delete(p.byPei, ueCtx.pei)
	delete(p.byTmsi, ueCtx.tmsi)
	ueCtx.Infof("UeContext removed")
}
