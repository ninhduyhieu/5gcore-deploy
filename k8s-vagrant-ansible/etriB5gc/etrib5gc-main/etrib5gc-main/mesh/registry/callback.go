package registry

import (
	"fmt"
	sbimodels "github.com/reogac/sbi/models"
	"sync/atomic"
)

func (reg *Registry) GetEndpointInfo() sbimodels.EndpointInfo {
	return sbimodels.EndpointInfo{
		Uuid: reg.id,
		//		GwId: reg.gwId,
	}
}

func (reg *Registry) GetEndpoint(epInfo *sbimodels.EndpointInfo) (ep *Endpoint, err error) {
	if atomic.LoadInt32(&reg.registered) != 1 {
		return nil, ErrNotRegistered
	}
	log.Tracef("Get endpoint %s", epInfo.Uuid)
	if len(epInfo.Uuid) == 0 {
		return nil, fmt.Errorf("No UUID for endpoint")
	}
	ep = reg.epman.createEndpointWithUuid(epInfo.Uuid)
	return
}
