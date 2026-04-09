package controller

import (
	"etrib5gc/mesh/models"
	"fmt"
	"github.com/reogac/utils/idgen"
	"math"
	"sync"
)

type ServiceGroup struct {
	service   string    //service identity
	name      string    //unique within a service
	idStr     string    //globally unique
	id        uint16    //generated
	selectors Selectors //selectors to select enpoints within the group
}

func (sg *ServiceGroup) setId(id uint16) {
	sg.id = id
	sg.idStr = fmt.Sprintf("%s-%s:%d:", sg.service, sg.name, sg.id)
}

type ServiceGrouping struct {
	service string                   //service id
	list    map[string]*ServiceGroup //indexed by group name(unique within a service)
}

type GroupList struct {
	services   map[string]*ServiceGrouping //index by service id
	groupIdGen idgen.Generator[uint16]
	groups     map[uint16]*ServiceGroup
	mutex      sync.RWMutex
}

func newGroupList() GroupList {
	return GroupList{
		services:   make(map[string]*ServiceGrouping),
		groups:     make(map[uint16]*ServiceGroup),
		groupIdGen: idgen.NewGenerator[uint16](0, math.MaxUint16),
	}
}
func (gl *GroupList) getGrouping(service string) (sg *ServiceGrouping) {
	gl.mutex.Lock()
	defer gl.mutex.Unlock()
	sg, _ = gl.services[service]
	return
}

func (gl *GroupList) getGroupNames(service string) (names []string) {
	gl.mutex.RLock()
	defer gl.mutex.RUnlock()
	if grouping, ok := gl.services[service]; ok {
		for gname, _ := range grouping.list {
			names = append(names, gname)
		}
	}
	return
}

func (gl *GroupList) addGroup(sg *ServiceGroup) {
	gl.mutex.Lock()
	defer gl.mutex.Unlock()

	if s, ok := gl.services[sg.service]; !ok {
		//generate a globally unique Id for the group
		sg.setId(gl.groupIdGen.Allocate())
		gl.groups[sg.id] = sg
		gl.services[sg.service] = &ServiceGrouping{
			service: sg.service,
			list:    map[string]*ServiceGroup{sg.name: sg},
		}
	} else {
		if old, ok := s.list[sg.name]; ok {
			//update old one
			old.selectors = sg.selectors
		} else {
			//generate a globally unique Id for the group
			sg.setId(gl.groupIdGen.Allocate())
			gl.groups[sg.id] = sg
			s.list[sg.name] = sg
		}
	}
}
func (gl *GroupList) buildGroupInfo(service string) map[string]models.Selectors {
	gl.mutex.RLock()
	defer gl.mutex.RUnlock()
	list := make(map[string]models.Selectors)
	if s, ok := gl.services[service]; ok {
		for name, item := range s.list {
			list[name] = models.Selectors(item.selectors)
		}
	}
	return list
}

func (gl *GroupList) listGroups() (all []*ServiceGroup) {
	gl.mutex.RLock()
	defer gl.mutex.RUnlock()
	for _, g := range gl.groups {
		all = append(all, g)
	}
	return
}
