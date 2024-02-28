package client

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
)

type ClientContextOpt func(clientContext client.Context) client.Context

type FactoryOpt func(factory tx.Factory) tx.Factory

type User interface {
	KeyName() string
	FormattedAddress() string
}

type Broadcaster struct {
	// buf stores the output sdk.TxResponse when broadcast.Tx is invoked.
	buf *bytes.Buffer

	cl *ChainClient

	// factoryOptions is a slice of broadcast.FactoryOpt which enables arbitrary configuration of the tx.Factory.
	factoryOptions []FactoryOpt
	// clientContextOptions is a slice of broadcast.ClientContextOpt which enables arbitrary configuration of the client.Context.
	clientContextOptions []ClientContextOpt
}

// NewBroadcaster returns a instance of Broadcaster which can be used with broadcast.Tx to
// broadcast messages sdk messages.
func NewBroadcaster(cl *ChainClient) *Broadcaster {
	return &Broadcaster{
		cl:  cl,
		buf: &bytes.Buffer{},
	}
}

// ConfigureFactoryOptions ensure the given configuration functions are run when calling GetFactory
// after all default options have been applied.
func (b *Broadcaster) ConfigureFactoryOptions(opts ...FactoryOpt) {
	b.factoryOptions = append(b.factoryOptions, opts...)
}

// ConfigureClientContextOptions ensure the given configuration functions are run when calling GetClientContext
// after all default options have been applied.
func (b *Broadcaster) ConfigureClientContextOptions(opts ...ClientContextOpt) {
	b.clientContextOptions = append(b.clientContextOptions, opts...)
}

// GetFactory returns an instance of tx.Factory that is configured with this Broadcaster's CosmosChain
// and the provided user. ConfigureFactoryOptions can be used to specify arbitrary options to configure the returned
// factory.
func (b *Broadcaster) GetFactory(ctx context.Context, user User) (tx.Factory, error) {
	clientContext, err := b.GetClientContext(ctx, user)
	if err != nil {
		return tx.Factory{}, err
	}

	sdkAdd, err := sdk.AccAddressFromBech32(user.FormattedAddress())
	if err != nil {
		return tx.Factory{}, err
	}

	account, err := clientContext.AccountRetriever.GetAccount(clientContext, sdkAdd)
	if err != nil {
		return tx.Factory{}, err
	}

	f := b.defaultTxFactory(clientContext, account)
	for _, opt := range b.factoryOptions {
		f = opt(f)
	}
	return f, nil
}

// GetClientContext returns a client context that is configured with this Broadcaster's CosmosChain and
// the provided user. ConfigureClientContextOptions can be used to configure arbitrary options to configure the returned
// client.Context.
func (b *Broadcaster) GetClientContext(ctx context.Context, user User) (client.Context, error) {

	sdkAdd, err := sdk.AccAddressFromBech32(user.FormattedAddress())
	if err != nil {
		return client.Context{}, err
	}

	clientContext := b.defaultClientContext(user, sdkAdd)
	for _, opt := range b.clientContextOptions {
		clientContext = opt(clientContext)
	}
	return clientContext, nil
}

// GetTxResponseBytes returns the sdk.TxResponse bytes which returned from broadcast.Tx.
func (b *Broadcaster) GetTxResponseBytes(ctx context.Context, user User) ([]byte, error) {
	if b.buf == nil || b.buf.Len() == 0 {
		return nil, fmt.Errorf("empty buffer, transaction has not been executed yet")
	}
	return b.buf.Bytes(), nil
}

// UnmarshalTxResponseBytes accepts the sdk.TxResponse bytes and unmarshalls them into an
// instance of sdk.TxResponse.
func (b *Broadcaster) UnmarshalTxResponseBytes(ctx context.Context, bytes []byte) (sdk.TxResponse, error) {
	resp := sdk.TxResponse{}
	if err := b.cl.Codec.Marshaler.UnmarshalJSON(bytes, &resp); err != nil {
		return sdk.TxResponse{}, err
	}
	return resp, nil
}

// defaultClientContext returns a default client context configured with the user as the sender.
func (b *Broadcaster) defaultClientContext(fromUser User, sdkAdd sdk.AccAddress) client.Context {
	// initialize a clean buffer each time
	b.buf.Reset()
	return b.cl.CliContext().
		WithOutput(b.buf).
		WithFrom(fromUser.FormattedAddress()).
		WithFromAddress(sdkAdd).
		WithFromName(fromUser.KeyName()).
		WithSkipConfirmation(true).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithKeyring(b.cl.Keybase).
		WithBroadcastMode(flags.BroadcastSync).
		WithCodec(b.cl.Codec.Marshaler)
}

// defaultTxFactory creates a new Factory with default configuration.
func (b *Broadcaster) defaultTxFactory(clientCtx client.Context, account client.Account) tx.Factory {
	chainConfig := b.cl.Config
	return tx.Factory{}.
		WithAccountNumber(account.GetAccountNumber()).
		WithSequence(account.GetSequence()).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithGasAdjustment(chainConfig.GasAdjustment).
		WithGas(flags.DefaultGasLimit).
		WithGasPrices(chainConfig.GasPrices).
		WithMemo("interchaintest").
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithKeybase(clientCtx.Keyring).
		WithChainID(clientCtx.ChainID).
		WithSimulateAndExecute(true)
}

// BroadcastTx uses the provided Broadcaster to broadcast all the provided messages which will be signed
// by the User provided. The sdk.TxResponse and an error are returned.
func BroadcastTx(ctx context.Context, broadcaster *Broadcaster, broadcastingUser User, msgs ...sdk.Msg) (sdk.TxResponse, error) {
	f, err := broadcaster.GetFactory(ctx, broadcastingUser)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	cc, err := broadcaster.GetClientContext(ctx, broadcastingUser)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	if err := tx.BroadcastTx(cc, f, msgs...); err != nil {
		return sdk.TxResponse{}, err
	}

	txBytes, err := broadcaster.GetTxResponseBytes(ctx, broadcastingUser)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	err = testutil.WaitForCondition(time.Second*30, time.Second*5, func() (bool, error) {
		var err error
		txBytes, err = broadcaster.GetTxResponseBytes(ctx, broadcastingUser)

		if err != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return sdk.TxResponse{}, err
	}

	respWithTxHash, err := broadcaster.UnmarshalTxResponseBytes(ctx, txBytes)
	if err != nil {
		return sdk.TxResponse{}, err
	} else if respWithTxHash.Code != 0 {
		return sdk.TxResponse{}, fmt.Errorf("TX hash %s got unexpected error code %d", respWithTxHash.TxHash, respWithTxHash.Code)
	}

	return getFullyPopulatedResponse(cc, respWithTxHash.TxHash)
}

// getFullyPopulatedResponse returns a fully populated sdk.TxResponse.
// the QueryTx function is periodically called until a tx with the given hash
// has been included in a block.
func getFullyPopulatedResponse(cc client.Context, txHash string) (sdk.TxResponse, error) {
	var resp sdk.TxResponse
	err := testutil.WaitForCondition(time.Second*60, time.Second*5, func() (bool, error) {
		fullyPopulatedTxResp, err := authtx.QueryTx(cc, txHash)
		if err != nil {
			return false, nil
		}

		resp = *fullyPopulatedTxResp
		return true, nil
	})
	return resp, err
}
