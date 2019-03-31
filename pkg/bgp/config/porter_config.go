package config

type PorterConfig struct {
	UsingPortForward bool `json:"using-port-forward,omitempty" mapstructure:"using-port-forward"`
}

func (p *PorterConfig) Equal(e *PorterConfig) bool {
	if p.UsingPortForward != e.UsingPortForward {
		return false
	}
	return true
}
