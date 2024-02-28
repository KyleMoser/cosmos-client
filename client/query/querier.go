package query

import (
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/strangelove-ventures/cosmos-client/client"
)

type Query struct {
	Client  *client.ChainClient
	Options *QueryOptions
}

// Bank queries

// Return params for bank module.
func (q *Query) Bank_Params() (*bankTypes.QueryParamsResponse, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return bank_ParamsRPC(q)
}

// Balances returns the balance of specific denom for a single account.
func (q *Query) Bank_Balance(address string, denom string) (*bankTypes.QueryBalanceResponse, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return bank_BalanceRPC(q, address, denom)
}

// Balances returns the balance of all coins for a single account.
func (q *Query) Bank_Balances(address string) (*bankTypes.QueryAllBalancesResponse, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return bank_AllBalancesRPC(q, address)
}

// SupplyOf returns the supply of given coin
func (q *Query) Bank_SupplyOf(denom string) (*bankTypes.QuerySupplyOfResponse, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return bank_SupplyOfRPC(q, denom)
}

// TotalSupply returns the supply of all coins
func (q *Query) Bank_TotalSupply() (*bankTypes.QueryTotalSupplyResponse, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return bank_TotalSupplyRPC(q)
}

// DenomMetadata returns the metadata for given denoms
func (q *Query) Bank_DenomMetadata(denom string) (*bankTypes.QueryDenomMetadataResponse, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return bank_DenomMetadataRPC(q, denom)
}

// DenomsMetadata returns the metadata for all denoms
func (q *Query) Bank_DenomsMetadata() (*bankTypes.QueryDenomsMetadataResponse, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return bank_DenomsMetadataRPC(q)
}

// Tendermint queries

// Block returns information about a block
func (q *Query) Block() (*coretypes.ResultBlock, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return BlockRPC(q)
}

// BlockByHash returns information about a block by hash
func (q *Query) BlockByHash(hash string) (*coretypes.ResultBlock, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return BlockByHashRPC(q, hash)
}

// BlockResults returns information about a block by hash
func (q *Query) BlockResults() (*coretypes.ResultBlockResults, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return BlockResultsRPC(q)
}

// Status returns information about a node status
func (q *Query) Status() (*coretypes.ResultStatus, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return StatusRPC(q)
}

// ABCIInfo returns general information about the ABCI application
func (q *Query) ABCIInfo() (*coretypes.ResultABCIInfo, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return ABCIInfoRPC(q)
}

// ABCIQuery returns data from a particular path in the ABCI application
func (q *Query) ABCIQuery(path string, data string, prove bool) (*coretypes.ResultABCIQuery, error) {
	/// TODO: In the future have some logic to route the query to the appropriate client (gRPC or RPC)
	return ABCIQueryRPC(q, path, data, prove)
}
