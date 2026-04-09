package models

type Labels map[string]string

type Selectors map[string]string

type Endpoint struct {
	Id        string
	InsId     string
	SbiUrl    string
	SbiPrefix string
	Labels    Labels
	GwId      string
	Name      string //server name for certificate verification
	Config    []byte
}

type RouteMatch map[string]string

type Destination struct {
	GroupId string `json:"group"`
	Weight  int    `json:"weight"`
	Lb      string `json:"lb"`
}

type Route struct {
	Match        RouteMatch    `json:"match"`        //how to match
	Destinations []Destination `json:"destinations"` //where to send requests
}

type Service struct {
	Id        string               `json:"id"` //service name
	Stateful  bool                 `json:"stateful"`
	Selectors Selectors            `json:"selectors"` //for selecting service endpoints
	Groups    map[string]Selectors `json:"groups"`    //pre-defined groups  of endpoints
	Routes    []Route              `json:"routes"`    //for route matching
}
type GatewayInfo struct {
	Id       string
	Url      string
	CertName string
}
