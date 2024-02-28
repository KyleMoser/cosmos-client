package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/strangelove-ventures/cosmos-client/client"
	"go.uber.org/zap"
)

// appState is the modifiable state of the application.
type appState struct {
	// Log is the root logger of the application.
	// Consumers are expected to store and use local copies of the logger
	// after modifying with the .With method.
	Log *zap.Logger

	Viper *viper.Viper

	HomePath        string
	OverriddenChain string
	Debug           bool
	Config          *configService
	cl              map[string]*client.ChainClient
}

// OverwriteConfig overwrites the config files on disk with the serialization of cfg,
// and it replaces a.Config with cfg.
//
// It is possible to use a brand new Config argument,
// but typically the argument is a.Config.
func (a *appState) OverwriteConfig(cfg *configService) error {
	home := a.Viper.GetString("home")
	cfgPath := path.Join(home, "config.yaml")
	if err := os.WriteFile(cfgPath, a.Config.MustYAML(), 0600); err != nil {
		return err
	}

	a.Config = cfg
	return nil
}

func (a *appState) Initialize(home string, logger *zap.Logger, cmd *cobra.Command, o map[string]ClientOverrides) error {
	a.cl = make(map[string]*client.ChainClient)
	for name, chain := range a.GetChainConfigs() {
		chain.Modules = append([]module.AppModuleBasic{}, ModuleBasics...)
		cl, err := client.NewChainClient(
			logger.With(zap.String("chain", name)),
			chain,
			home,
			cmd.InOrStdin(),
			cmd.OutOrStdout(),
		)
		if err != nil {
			return fmt.Errorf("error creating chain client: %w", err)
		}
		// If overrides are present (should only happen in test), modify the client to use those overrides.
		if o != nil {
			if rc := o[name].RPCClient; rc != nil {
				cl.RPCClient = rc
			}
			if lp := o[name].LightProvider; lp != nil {
				cl.LightProvider = lp
			}
		}
		a.cl[name] = cl
	}

	// override chain if needed
	if cmd.PersistentFlags().Changed("chain") {
		defaultChain, err := cmd.PersistentFlags().GetString("chain")
		if err != nil {
			return err
		}

		a.Config.clientConfig.SetDefaultChain(defaultChain)
	}

	if cmd.PersistentFlags().Changed("output") {
		output, err := cmd.PersistentFlags().GetString("output")
		if err != nil {
			return err
		}

		for _, chain := range a.GetChainConfigs() {
			chain.OutputFormat = output
		}
	}

	return nil
}

func (a *appState) GetChainConfigs() map[string]*client.ChainClientConfig {
	if a.Config.clientConfig == nil {
		return nil
	}
	return a.Config.clientConfig.GetChainConfigs()
}

func (a *appState) GetDefaultClient() *client.ChainClient {
	if a.Config.clientConfig == nil {
		return nil
	}
	return a.GetClient(a.Config.clientConfig.GetDefaultChain())
}

func (a *appState) GetClient(chainID string) *client.ChainClient {
	if v, ok := a.cl[chainID]; ok {
		return v
	}
	return nil
}
