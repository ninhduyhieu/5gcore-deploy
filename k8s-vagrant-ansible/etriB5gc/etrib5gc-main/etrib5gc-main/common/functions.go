package common

import (
	"fmt"
	"github.com/reogac/sbi/models"
	"net"
	"strconv"
	"strings"
)

func IsSliceEqual(s1, s2 *models.Snssai) bool {
	if s1 == nil && s2 == nil {
		return true
	} else if s1 == nil || s2 == nil {
		return false
	}

	return s1.Sst == s2.Sst && strings.Compare(s1.Sd, s2.Sd) == 0
}

func IsPlmnIdEqual(id1, id2 *models.PlmnId) bool {
	if id1 == nil && id2 == nil {
		return true
	} else if id1 == nil || id2 == nil {
		return false
	}

	return strings.Compare(id1.Mnc, id2.Mnc) == 0 && strings.Compare(id1.Mcc, id2.Mcc) == 0
}
func ServingNetworkName(id *models.PlmnId) string {
	//return fmt.Sprintf("5G:mnc%03x.mcc%03x.3gppnetwork.org", id.Mnc, id.Mcc)
	if len(id.Mnc) == 2 {
		return fmt.Sprintf("5G:mnc0%s.mcc%s.3gppnetwork.org", id.Mnc, id.Mcc)
	} else {
		return fmt.Sprintf("5G:mnc%s.mcc%s.3gppnetwork.org", id.Mnc, id.Mcc)
	}
}

// create a map from a list
func MakeSet[T comparable](items []T) (m map[T]struct{}) {
	m = make(map[T]struct{})
	for _, t := range items {
		if _, ok := m[t]; !ok {
			m[t] = struct{}{}
		}
	}
	return
}

type Stringer interface {
	String() string
}

// create a string-indexed map from a list of Stringer items
func MakeStringSet[T Stringer](items []T) (m map[string]T) {
	m = make(map[string]T)
	for _, t := range items {
		if _, ok := m[t.String()]; !ok {
			m[t.String()] = t
		}
	}
	return
}

// make a list of unique item from a list
func UniqueArray[T comparable](items []T) (ret []T) {
	itemset := MakeSet[T](items)
	for i, _ := range itemset {
		ret = append(ret, i)
	}
	return
}

func L4Address(ip net.IP, port int) (addr string) {
	if len(ip) > 0 {
		addr = fmt.Sprintf("%s:%d", ip.String(), port)
	}
	return
}

func ValuePointer[T any](v T) *T {
	return &v
}

func BitRate2kbps(bitrate string) uint64 {
	s := strings.Split(bitrate, " ")
	var kbps uint64

	var digit int

	if n, err := strconv.Atoi(s[0]); err != nil {
		return 0
	} else {
		digit = n
	}

	switch s[1] {
	case "bps":
		kbps = uint64(digit / 1000)
	case "Kbps":
		kbps = uint64(digit * 1)
	case "Mbps":
		kbps = uint64(digit * 1000)
	case "Gbps":
		kbps = uint64(digit * 1000000)
	case "Tbps":
		kbps = uint64(digit * 1000000000)
	}
	return kbps
}
