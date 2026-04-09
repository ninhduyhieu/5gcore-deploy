package mesh

const (
	SBI_PORT       string = "9001"
	AGENT_PORT     string = "7001"
	REGISTRAR_PORT string = "7777"
)

type MeshConfig struct {
	Bind                string            `json:"bind"`
	AgentPort           *int              `json:"agentPort,omitempty"`
	Registrar           string            `json:"registrar"`
	RegisteredSbiUrl    string            `json:"registeredSbiUrl"`
	RegisteredAgentPort *int              `json:"registeredAgentPort"`
	Labels              map[string]string `json:"labels,omitempty"`
}
