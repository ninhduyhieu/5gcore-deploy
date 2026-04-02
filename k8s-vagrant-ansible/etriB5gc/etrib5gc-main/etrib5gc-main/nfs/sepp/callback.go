package sepp

import (
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi/models"
	"strings"
)

type CallbackInfo struct {
	SeppId string //SEPP's url
	NfId   string
}

func (nf *SEPP) rewriteEgressCallback(old string) (string, error) {
	//from uuid[gw] to [sepp url + nfId]
	epInfo, err := models.EndpointInfoFromString(old)
	if err != nil {
		return "", fmt.Errorf("Invalid callback value: %+v", err)
	}
	cli, err := mesh.ConsumerFromEndpoint(epInfo)
	if err != nil {
		return "", fmt.Errorf("NF is unreachable: %+v", err)
	}
	//add new LocalEndpoint
	ep := &LocalEndpoint{
		cli: cli,
	}
	nf.localEndpoints.add(ep)

	newCallback := fmt.Sprintf("%s;;%s", nf.url, ep.nfId)
	nf.Infof("Rewrite Egress callback from %v to %s", *epInfo, newCallback)
	return newCallback, nil
}

func (nf *SEPP) rewriteIngressCallback(old string) (string, error) {
	//from [remote sepp url + nfid] to [local sepp EndpointInfo + localId]
	parts := strings.Split(old, ";;")
	if len(parts) != 2 {
		return "", fmt.Errorf("Invalid callback value")
	}

	epInfo := mesh.EndpointInfo()
	seppId := parts[0]
	nfId := parts[1]

	remoteId := remoteId(seppId, nfId)
	var ep *RemoteEndpoint
	if ep = nf.remoteEndpoints.findByRemoteId(remoteId); ep == nil {
		if peer := nf.peers.getPeer(seppId); peer == nil {
			return "", fmt.Errorf("Invalid remote SEPP")
		} else {
			ep = &RemoteEndpoint{
				sepp: peer,
				nfId: nfId,
			}
			nf.remoteEndpoints.add(ep)
			nf.Infof("Create remote endpoint: sepp=%s, nfId=%s, localId=%s", peer.url, ep.nfId, ep.localId)
		}
	}
	epInfo.Stickiness = ep.localId
	nf.Infof("Rewrite callback to :%v", epInfo)
	//add new RemoteEndpoint
	newCallback := models.EndpointInfoToString(*epInfo)
	return newCallback, nil
}
