package lbs

import "strings"

const (
	LB_TYPE_RANDOM LoadBalancerType = iota
	LB_TYPE_ROUNDROBIN
	LB_TYPE_UNKNOWN
)

type LoadBalancerType uint8

func ParseLoadBalancerType(s string) LoadBalancerType {
	switch strings.ToLower(s) {
	case "random":
		return LB_TYPE_RANDOM
	case "roundrobin":
		return LB_TYPE_ROUNDROBIN
	default:
	}
	return LB_TYPE_UNKNOWN
}

func (lb LoadBalancerType) String() string {
	switch lb {
	case LB_TYPE_RANDOM:
		return "Random"
	case LB_TYPE_ROUNDROBIN:
		return "RoundRobin"
	}
	return "Unknown"
}

type Endpoint interface {
	Id() string
}

type LoadBalancer interface {
	AddEndpoint(Endpoint)
	RemoveEndpoint(Endpoint)
	PickEndpoint() Endpoint
	NumEndpoints() int
}

func CreateLoadBalancer(t LoadBalancerType) LoadBalancer {
	switch t {
	case LB_TYPE_RANDOM:
		return createRandomLB()
	case LB_TYPE_ROUNDROBIN:
		return createRandomLB()
	default:
		return createRandomLB()
	}
}
