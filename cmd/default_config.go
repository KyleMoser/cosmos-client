package cmd

import (
	"fmt"
	"path"

	"github.com/KyleMoser/cosmos-client/client"
)

var _ configPart = (*CosmosClientConfig)(nil)

// DefaultConfig represents the config file for Cosmos chains
type CosmosClientConfig struct {
	DefaultChain string                               `yaml:"default_chain" json:"default_chain"`
	Chains       map[string]*client.ChainClientConfig `yaml:"chains" json:"chains"`
	Debug        bool
}

func (c *CosmosClientConfig) SetChainConfig(name string, config *client.ChainClientConfig) {
	c.Chains[name] = config
}

func (c *CosmosClientConfig) GetChainConfigs() map[string]*client.ChainClientConfig {
	return c.Chains
}

func (c *CosmosClientConfig) SetDefaultChain(chain string) {
	c.DefaultChain = chain
}
func (c *CosmosClientConfig) GetDefaultChain() string {
	return c.DefaultChain
}

func (c *CosmosClientConfig) ValidateConfig() error {
	for _, chain := range c.Chains {
		if err := chain.Validate(); err != nil {
			return err
		}
	}
	if c.GetDefaultChain() == "" {
		return fmt.Errorf("default chain (%s) configuration not found", c.DefaultChain)
	}
	return nil
}

func (c *CosmosClientConfig) CreateNewConfig(home string) {
	keyHome := path.Join(home, "keys")
	c.DefaultChain = "cosmoshub"
	c.Chains = map[string]*client.ChainClientConfig{}
	c.Chains["cosmoshub"] = client.GetCosmosHubConfig(keyHome, c.Debug)
	c.Chains["osmosis"] = client.GetOsmosisConfig(keyHome, c.Debug)
}
