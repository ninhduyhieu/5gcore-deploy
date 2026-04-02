package sepp

import (
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/httpw"
	"sync"
)

type SeppPeer struct {
	cli *httpw.Client
	url string
}

func (p *SeppPeer) getId() string {
	return p.url
}

type PeerList struct {
	mutex      sync.RWMutex
	peers      map[models.PlmnId][]*SeppPeer
	urlIndexes map[string]*SeppPeer
}

func newPeerList(infos []models.HomePlmnConfiguration) *PeerList {
	peers := make(map[models.PlmnId][]*SeppPeer)
	indexes := make(map[string]*SeppPeer)
	for _, item := range infos {
		list, _ := peers[item.Id]
		for _, addr := range item.Sepps {
			p := &SeppPeer{
				url: addr,
				cli: httpw.NewClient(nil, nil, ""),
			}
			indexes[addr] = p
			list = append(list, p)
		}
		peers[item.Id] = list
	}
	return &PeerList{
		peers:      peers,
		urlIndexes: indexes,
	}
}

func (l *PeerList) getPeer(url string) *SeppPeer {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	p, _ := l.urlIndexes[url]
	return p
}

func (l *PeerList) pickPeer(plmnId models.PlmnId) *SeppPeer {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if list, ok := l.peers[plmnId]; ok {
		//TODO: pick random
		return list[0]
	}
	return nil
}
