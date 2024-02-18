package cmd

import (
	feegrant "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	wasm "github.com/CosmWasm/wasmd/x/wasm"
	cosmosmodule "github.com/KyleMoser/cosmos-client/cmd/modules"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	authz "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/ibc-go/modules/capability"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
)

// TODO: Import a bunch of custom modules like cosmwasm and osmosis
// Problem is SDK versioning. Need to find a fix for this.

var ModuleBasics = []module.AppModuleBasic{
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
	wasm.AppModuleBasic{},
	capability.AppModuleBasic{},
	cosmosmodule.AppModuleBasic{},
	vesting.AppModuleBasic{},
}
