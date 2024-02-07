package client

import (
	"context"
	"fmt"
	"time"

	registry "github.com/KyleMoser/cosmos-client/client/chain_registry"

	feegrant "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authz "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	ModuleBasics = []module.AppModuleBasic{
		auth.AppModuleBasic{},
		authz.AppModuleBasic{},
		bank.AppModuleBasic{},
		distribution.AppModuleBasic{},
		feegrant.AppModuleBasic{},
		params.AppModuleBasic{},
		slashing.AppModuleBasic{},
		staking.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ibc.AppModuleBasic{},
	}
)

type ChainClientConfig struct {
	ChainName      string                  `json:"-" yaml:"-"`
	Key            string                  `json:"key" yaml:"key"`
	ChainID        string                  `json:"chain-id" yaml:"chain-id"`
	RPCAddr        string                  `json:"rpc-addr" yaml:"rpc-addr"`
	GRPCAddr       string                  `json:"grpc-addr" yaml:"grpc-addr"`
	AccountPrefix  string                  `json:"account-prefix" yaml:"account-prefix"`
	KeyringBackend string                  `json:"keyring-backend" yaml:"keyring-backend"`
	GasAdjustment  float64                 `json:"gas-adjustment" yaml:"gas-adjustment"`
	GasPrices      string                  `json:"gas-prices" yaml:"gas-prices"`
	MinGasAmount   uint64                  `json:"min-gas-amount" yaml:"min-gas-amount"`
	KeyDirectory   string                  `json:"key-directory" yaml:"key-directory"`
	Debug          bool                    `json:"debug" yaml:"debug"`
	Timeout        string                  `json:"timeout" yaml:"timeout"`
	BlockTimeout   string                  `json:"block-timeout" yaml:"block-timeout"`
	OutputFormat   string                  `json:"output-format" yaml:"output-format"`
	SignModeStr    string                  `json:"sign-mode" yaml:"sign-mode"`
	ExtraCodecs    []string                `json:"extra-codecs" yaml:"extra-codecs"`
	Modules        []module.AppModuleBasic `json:"-" yaml:"-"`
	Slip44         int                     `json:"slip44" yaml:"slip44"`
}

func (ccc *ChainClientConfig) Validate() error {
	if _, err := time.ParseDuration(ccc.Timeout); err != nil {
		return err
	}
	if ccc.BlockTimeout != "" {
		if _, err := time.ParseDuration(ccc.BlockTimeout); err != nil {
			return err
		}
	}
	return nil
}

func GetCosmosHubConfig(keyHome string, debug bool) *ChainClientConfig {
	return &ChainClientConfig{
		Key:            "default",
		ChainID:        "cosmoshub-4",
		RPCAddr:        "https://cosmoshub-4.technofractal.com:443",
		GRPCAddr:       "https://gprc.cosmoshub-4.technofractal.com:443",
		AccountPrefix:  "cosmos",
		KeyringBackend: "test",
		GasAdjustment:  1.2,
		GasPrices:      "0.01uatom",
		MinGasAmount:   0,
		KeyDirectory:   keyHome,
		Debug:          debug,
		Timeout:        "20s",
		OutputFormat:   "json",
		SignModeStr:    "direct",
	}
}

func GetOsmosisConfig(keyHome string, debug bool) *ChainClientConfig {
	return &ChainClientConfig{
		Key:            "default",
		ChainID:        "osmosis-1",
		RPCAddr:        "https://osmosis-1.technofractal.com:443",
		GRPCAddr:       "https://gprc.osmosis-1.technofractal.com:443",
		AccountPrefix:  "osmo",
		KeyringBackend: "test",
		GasAdjustment:  1.2,
		GasPrices:      "0.01uosmo",
		MinGasAmount:   0,
		KeyDirectory:   keyHome,
		Debug:          debug,
		Timeout:        "20s",
		OutputFormat:   "json",
		SignModeStr:    "direct",
	}
}

func GetChainConfigWithOpts(ctx context.Context, c registry.ChainInfo, opts *ChainConfigOptions) (*ChainClientConfig, error) {
	debug := viper.GetBool("debug")
	home := viper.GetString("home")

	assetList, err := c.GetAssetList()
	if err != nil {
		return nil, err
	}

	var gasPrices string
	if len(assetList.Assets) > 0 {
		gasPrices = fmt.Sprintf("%.2f%s", 0.01, assetList.Assets[0].Base)
	}

	var rpc string
	if opts != nil && len(opts.PreferredRpcHosts) > 0 {
		rpc, err = c.GetPreferredRPCEndpoint(ctx, opts.PreferredRpcHosts)
	}

	if err != nil {
		rpc, err = c.GetRandomRPCEndpoint(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &ChainClientConfig{
		ChainName:      c.ChainName,
		Key:            "default",
		ChainID:        c.ChainID,
		RPCAddr:        rpc,
		AccountPrefix:  c.Bech32Prefix,
		KeyringBackend: "test",
		GasAdjustment:  1.2,
		GasPrices:      gasPrices,
		KeyDirectory:   home,
		Debug:          debug,
		Timeout:        "20s",
		OutputFormat:   "json",
		SignModeStr:    "direct",
		Slip44:         c.Slip44,
	}, nil
}

type ChainConfigOptions struct {
	PreferredRpcHosts []string
}

func GetChain(ctx context.Context, chainName string, logger *zap.Logger, options *ChainConfigOptions) (*ChainClientConfig, error) {
	registry := registry.DefaultChainRegistry(logger)
	chainInfo, err := registry.GetChain(chainName)
	if err != nil {
		logger.Info(
			"Failed to get chain",
			zap.String("name", chainName),
			zap.Error(err),
		)
		return nil, err
	}

	return GetChainConfigWithOpts(ctx, chainInfo, options)
}
