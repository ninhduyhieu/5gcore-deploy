package oambackend

type EndpointInfo struct {
	RegTime            string
	Uuid               string
	GwId               string
	Labels             map[string]string
	OfferingServices   []string
	SubscribedServices []string
}

type GatewayInfo struct {
	Uuid      string
	Name      string
	RegTime   string
	LastSeen  string
	Url       string
	Labels    map[string]string
	Endpoints []string
}

type ServiceInfo struct {
	Name           string
	Selectors      map[string]string
	Stateful       bool
	NumEndpoints   int
	NumSubscribers int
	Routes         []int    //list of route id in a decreasing priority order
	Groups         []string //list of group name
}

type GroupInfo struct {
	Id        int
	IdStr     string
	Name      string
	Service   string
	Selectors map[string]string
}

type RouteInfo struct {
	Id           int
	IdStr        string
	Service      string
	Match        map[string]string
	Destinations []DestinationInfo
}

type DestinationInfo struct {
	GroupId      string
	Weight       int
	LoadBalancer string
}
