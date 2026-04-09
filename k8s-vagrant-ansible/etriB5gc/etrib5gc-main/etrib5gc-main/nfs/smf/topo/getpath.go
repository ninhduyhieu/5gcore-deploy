package topo

import (
	"etrib5gc/nfs/smf/dataplane"
	"etrib5gc/util/dijkstra"
	"fmt"
	"math/rand"
	"net"
	"strings"
)

// query to find UPF path
type UpfQuery struct {
	TransportNetworks []string
	Dnai              string //Note: not use for now
	Dnn               string
}

func (q *UpfQuery) String() string {
	return fmt.Sprintf("%s-%s", strings.Join(q.TransportNetworks, ","), q.Dnai)
}

// A PathNode represents a node in UPF path of a tunnel
// it has uplink, downlink face IP and the sbi address of the upf
type PathNode struct {
	UlIp, DlIp net.IP
	Upf        *dataplane.Upf
}

// One data path
type UpfPath []PathNode

// Find an Upf path and allocate UeIp
func (t *upfTopoMan) getPath(query UpfQuery) (datapath UpfPath, err error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	//find all anchors and source nodes for searching(at the same time)
	log.Debugf("Msg query: nets = %s", strings.Join(query.TransportNetworks, ", "))
	log.Debugf("Number of UPF node: %d", len(t.nodes))

	dstFaces := make(map[string][]NetInf) //faces of anchoring upfs
	anchors := []*UpfNode{}               //keep all anchoring nodes
	allNodes := []*UpfNode{}              //array of all nodes

	srcFaces := []NetInf{} //faces to start searching
	for _, node := range t.nodes {
		allNodes = append(allNodes, node)
		if node.hasDnn(query.Dnn) {
			anchors = append(anchors, node)
			dstFaces[node.id] = node.getInfs()
		}

		if infs := node.getFacesForNets(query.TransportNetworks); len(infs) > 0 { //a starting node
			srcFaces = append(srcFaces, infs...)
		}
	}

	if len(anchors) < 1 {
		err = fmt.Errorf("Cannot find any anchoring UPF")
		return
	} else if len(srcFaces) < 1 {
		err = fmt.Errorf("Cannot find any srcFaces")
		return
	}

	//select an anchor
	var anchor *UpfNode //selected anchoring node
	//	1. randomize the anchor list
	for i, _ := range anchors {
		j := rand.Intn(i + 1)
		anchors[i], anchors[j] = anchors[j], anchors[i]
	}
	//pick the first then allocate IP
	anchor = anchors[0]
	dstface := anchor.pickRandomFace() //take the first netinf from this node as destination face

	//build a graph of active links then find the shortest paths from source to
	//destination
	edges := []dijkstra.EdgeInfo{} //edges to build the grap

	//create edges between every netinf between nodes in a same network
	for i := 0; i < len(allNodes)-1; i++ {
		node := allNodes[i]
		for j := i + 1; j < len(allNodes); j++ {
			peer := allNodes[j]
			for netname, infs1 := range node.infs {
				if infs2, ok := peer.infs[netname]; ok {
					for _, f1 := range infs1 {
						for _, f2 := range infs2 {
							edges = append(edges, dijkstra.EdgeInfo{
								A: f1.id,
								B: f2.id,
								W: 10,
							})
						}
					}
				}
			}
		}
	}
	//create edges between internal netinfs of a node
	for _, node := range allNodes {
		edges = append(edges, node.internalEdges()...)
	}

	graph := dijkstra.New(edges)
	//	1. shuffle sources
	for i := range srcFaces {
		j := rand.Intn(i + 1)
		srcFaces[i], srcFaces[j] = srcFaces[j], srcFaces[i]
	}
	for _, srcface := range srcFaces {
		log.Tracef("Search path from %s to %s", srcface.id, dstface.id)
		if _, paths := graph.ShortestPath(srcface.id, dstface.id); len(paths) > 0 {
			path := paths[0] //pick the first path
			log.Tracef("Found path: %v", path)
			plen := len(path)

			i := 0
			pathnodes := []PathNode{}
			for i < plen {
				dlface := t.faces[path[i]]
				node := PathNode{
					DlIp: dlface.ip,
					Upf:  dlface.local.upf,
				}
				pathnodes = append(pathnodes, node)
				i++
				if i > plen-1 {
					break
				}
				ulface := t.faces[path[i]]
				node.UlIp = ulface.ip
				i++
			}
			//pathnodes[plen-1].UlIp = pathnodes[plen-1].DlIp
			datapath = pathnodes
			break
		}
	}
	if datapath == nil || len(datapath) < 1 {
		err = fmt.Errorf("Cannot find path")
	}

	return
}
