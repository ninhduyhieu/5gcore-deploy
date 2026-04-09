package dijkstra

import (
	"container/heap"
	"fmt"
	"math"
	"strings"
)

const (
	DIST_INF int64 = math.MaxInt64
)

type EdgeInfo struct {
	A string
	B string
	W int64
}

type PQ []*Node

func (pq PQ) Len() int           { return len(pq) }
func (pq PQ) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq PQ) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PQ) Push(x any) {
	node := x.(*Node)
	node.index = len(*pq)
	*pq = append(*pq, node)
}

func (pq *PQ) Pop() any {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}

type Edge struct {
	node *Node
	dist int64
}
type Node struct {
	id      string
	edges   []*Edge //edges to neighbors
	visited bool
	dist    int64      //current shortest distance from the source
	paths   [][]string //current shortest paths from the source
	index   int
}

func (n *Node) addEdge(other *Node, w int64) {
	found := false
	for _, e := range n.edges {
		if strings.Compare(e.node.id, other.id) == 0 {
			found = true
		}
	}
	if !found {
		n.edges = append(n.edges, &Edge{
			node: other,
			dist: w,
		})
	}
}
func (n *Node) String() string {
	return fmt.Sprintf("%s:%d:%v", n.id, n.dist, n.paths)
}

type Graph struct {
	nodes map[string]*Node //get node from its id
}

func New(edges []EdgeInfo) (g Graph) {
	g.nodes = make(map[string]*Node)
	var (
		a, b *Node
		ok   bool
	)

	for _, e := range edges {
		if a, ok = g.nodes[e.A]; !ok {
			a = &Node{
				id: e.A,
			}
			g.nodes[a.id] = a
		}
		if b, ok = g.nodes[e.B]; !ok {
			b = &Node{
				id: e.B,
			}
			g.nodes[b.id] = b
		}
		if a == b {
			continue
		}
		a.addEdge(b, e.W)
		b.addEdge(a, e.W)
	}
	return
}

func (graph *Graph) ShortestPath(s, d string) (dist int64, paths [][]string) {
	var (
		cur, source, destination *Node
		ok                       bool
		pq                       PQ
	)
	//source and desitination are identical
	if strings.Compare(s, d) == 0 {
		paths = [][]string{[]string{s}}
	}

	//otherewise they must be in the graph
	if source, ok = graph.nodes[s]; !ok {
		return
	}

	if destination, ok = graph.nodes[d]; !ok {
		return
	}

	for _, node := range graph.nodes {
		//reset nodes
		node.visited = false
		node.dist = DIST_INF
		node.index = -1
	}
	source.dist = 0
	source.paths = [][]string{[]string{source.id}}
	heap.Push(&pq, source)
	for len(pq) > 0 {
		cur = heap.Pop(&pq).(*Node)
		//	fmt.Printf("pop %s\n", cur)
		if cur == destination {
			paths = cur.paths
			dist = cur.dist
			break
		}

		if cur.visited {
			//		fmt.Printf("%s was visited\n", cur)
			continue
		}
		//	fmt.Printf("visit %s\n", cur)
		cur.visited = true
		for _, e := range cur.edges {
			if !e.node.visited {
				plen := cur.dist + e.dist
				if plen < e.node.dist { //update path distance
					e.node.dist = plen
					e.node.paths = make([][]string, len(cur.paths))
					for i, p := range cur.paths {
						e.node.paths[i] = make([]string, len(p)+1)
						copy(e.node.paths[i][:], p)
						e.node.paths[i][len(p)] = e.node.id
					}
					if e.node.index == -1 {
						//					fmt.Printf("push %s\n", e.node)
						heap.Push(&pq, e.node)
					} else {
						//					fmt.Printf("fix %s\n", e.node)
						heap.Fix(&pq, e.node.index)
					}

				} else if plen == e.node.dist {
					for _, p := range cur.paths {
						newpath := make([]string, len(p)+1)
						copy(newpath, p)
						newpath[len(p)] = e.node.id
						e.node.paths = append(e.node.paths, newpath)
					}
				}
			}
		}
	}
	return
}
