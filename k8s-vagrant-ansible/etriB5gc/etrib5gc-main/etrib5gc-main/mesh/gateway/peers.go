package gateway

import (
	"github.com/reogac/utils/httpw"
	"sync"
)

type PeerGw struct {
	id   string
	addr string
	cli  *httpw.Client
}

type Peers struct {
	peers map[string]*PeerGw
	mutex sync.RWMutex
}

func newPeers() *Peers {
	return &Peers{
		peers: make(map[string]*PeerGw),
	}
}

func (l *Peers) add(id, url string, cli *httpw.Client) *PeerGw {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	p := &PeerGw{
		id:   id,
		addr: url,
		cli:  cli,
	}
	l.peers[id] = p
	return p
}

func (l *Peers) remove(id string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	delete(l.peers, id)
}

func (l *Peers) get(id string) *PeerGw {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	p, _ := l.peers[id]
	return p
}
