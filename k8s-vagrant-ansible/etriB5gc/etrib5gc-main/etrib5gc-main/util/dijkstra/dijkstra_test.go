package dijkstra

import (
	"fmt"
	"strings"
	"testing"
)

type TestCase struct {
	s     string
	d     string
	dist  int64
	paths [][]string
}

func (tc *TestCase) pass(dist int64, paths [][]string) bool {
	if dist != tc.dist || len(paths) != len(paths) {
		return false
	}
	pathmap := make(map[string]bool)
	for _, path := range tc.paths {
		pathmap[strings.Join(path, ",")] = true
	}
	for _, path := range paths {
		if _, ok := pathmap[strings.Join(path, ",")]; !ok {
			return false
		}
	}
	return true
}

func Test_Dijkstra(t *testing.T) {
	edges := []EdgeInfo{
		EdgeInfo{
			A: "1",
			B: "2",
			W: 7,
		},
		EdgeInfo{
			A: "1",
			B: "3",
			W: 9,
		},
		EdgeInfo{
			A: "1",
			B: "6",
			W: 14,
		},
		EdgeInfo{
			A: "2",
			B: "3",
			W: 10,
		},
		EdgeInfo{
			A: "2",
			B: "4",
			W: 15,
		},
		EdgeInfo{
			A: "3",
			B: "6",
			W: 2,
		},
		EdgeInfo{
			A: "3",
			B: "4",
			W: 11,
		},
		EdgeInfo{
			A: "6",
			B: "5",
			W: 9,
		},
		EdgeInfo{
			A: "5",
			B: "4",
			W: 6,
		},
	}
	cases := []TestCase{
		TestCase{
			s:     "1",
			d:     "5",
			dist:  20,
			paths: [][]string{[]string{"1", "3", "6", "5"}},
		},
		TestCase{
			s:     "5",
			d:     "1",
			dist:  20,
			paths: [][]string{[]string{"5", "6", "3", "1"}},
		},
		TestCase{
			s:     "3",
			d:     "3",
			dist:  0,
			paths: [][]string{[]string{"3"}},
		},
		TestCase{
			s:     "6",
			d:     "3",
			dist:  2,
			paths: [][]string{[]string{"6", "3"}},
		},
		TestCase{
			s:     "2",
			d:     "5",
			dist:  21,
			paths: [][]string{[]string{"2", "4", "5"}, []string{"2", "3", "6", "5"}},
		},
	}
	graph := New(edges)
	var dist int64
	var paths [][]string
	for i, tc := range cases {
		dist, paths = graph.ShortestPath(tc.s, tc.d)
		if !tc.pass(dist, paths) {
			t.Errorf("test case %d is not passed: distance: %d(%d); paths: %v(%v)\n", i, dist, tc.dist, paths, tc.paths)
		} else {
			fmt.Printf("test case %d: distance: %d(%d); paths: %v(%v)\n", i, dist, tc.dist, paths, tc.paths)
		}
	}
}
