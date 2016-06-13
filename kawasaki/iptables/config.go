package iptables

type IPTablesConfig struct {
	PreroutingChain     string
	PostroutingChain    string
	InputChain          string
	ForwardChain        string
	DefaultChain        string
	InstanceChainPrefix string
}

func NewConfig(chainPrefix string) IPTablesConfig {
	return IPTablesConfig{
		PreroutingChain:     chainPrefix + "prerouting",
		PostroutingChain:    chainPrefix + "postrouting",
		InputChain:          chainPrefix + "input",
		ForwardChain:        chainPrefix + "forward",
		DefaultChain:        chainPrefix + "default",
		InstanceChainPrefix: chainPrefix + "instance-",
	}
}

func InstanceChain(config IPTablesConfig, instanceId string) string {
	return config.InstanceChainPrefix + instanceId
}
