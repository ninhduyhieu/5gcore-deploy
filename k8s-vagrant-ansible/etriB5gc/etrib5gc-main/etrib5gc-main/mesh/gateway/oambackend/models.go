package oambackend

type EndpointInfo struct {
	RegTime  string
	LastSeen string
	Uuid     string
	AgentUrl string
	SbiUrl   string
	Labels   map[string]string
}
