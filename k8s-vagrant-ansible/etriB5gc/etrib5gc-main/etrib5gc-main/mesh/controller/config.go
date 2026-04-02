package controller

import (
	"etrib5gc/mesh/lbs"
)

const (
	CONTROLLER_PORT string               = "8888"
	DEFAULT_LB      lbs.LoadBalancerType = lbs.LB_TYPE_RANDOM
)

type ServiceConfig struct {
	Id        string    `json:"id"`
	Selectors Selectors `json:"selectors"`
	Stateful  bool      `json:"stateful"`
}

type GroupConfig struct {
	Id        string    `json:"id"`
	Selectors Selectors `json:"selectors"`
}

type GroupingConfig struct {
	Service string        `json:"service"`
	Groups  []GroupConfig `json:"groups"`
}

type RouteMatchConfig map[string]string

type DestinationConfig struct {
	Group  string `json:"group"`
	Weight uint16 `json:"weight"`
	Lb     string `json:"lb"`
}

type RouteConfig struct {
	Match        RouteMatchConfig    `json:"match"`
	Destinations []DestinationConfig `json:"destinations"`
}

type RoutingConfig struct {
	Service string        `json:"service"`
	Routes  []RouteConfig `json:"routes"`
}

type Config struct {
	Bind      string           `json:"bind"`
	Services  []ServiceConfig  `json:"services"`
	Groupings []GroupingConfig `json:"groupings"`
	Routings  []RoutingConfig  `json:"routings"`
}
