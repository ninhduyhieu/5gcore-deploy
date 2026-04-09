package gateway

const (
	GATEWAY_PORT int = 7777
)

type Config struct {
	Bind           string `json:"bind"`
	RegisteredAddr string `json:"registeredAddr"`
	Controller     string `json:"controller"`

	Heartbeat int               `json:"heartbeat"`
	Labels    map[string]string `json:"labels"`
}
