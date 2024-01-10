package chain_registry

// Defines IBC connection details between Cosmos chains.
// From e.g. https://raw.githubusercontent.com/cosmos/chain-registry/master/_IBC/cosmoshub-osmosis.json
type IbcConfig struct {
	Schema string         `json:"$schema"`
	Chain1 IbcConfigChain `json:"chain_1"`
	Chain2 IbcConfigChain `json:"chain_2"`

	Channels []IbcConfigChannelOuter `json:"channels"`
}

type IbcConfigChannelOuter struct {
	Chain1   IbcConfigChannel `json:"chain_1"`
	Chain2   IbcConfigChannel `json:"chain_2"`
	Ordering string           `json:"ordering"`
	Version  string           `json:"version"`

	Tags struct {
		Status    string `json:"status,omitempty"`
		Preferred bool   `json:"preferred,omitempty"`
		Dex       string `json:"dex,omitempty"`
	} `json:"tags,omitempty"`
}

type IbcConfigChain struct {
	ChainName    string `json:"chain_name"`
	ClientId     string `json:"client_id"`
	ConnectionId string `json:"connection_id"`
}

type IbcConfigChannel struct {
	ChannelId string `json:"channel_id"`
	PortId    string `json:"port_id"`
}
