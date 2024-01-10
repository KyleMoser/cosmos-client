package client_test

import (
	"context"
	"testing"

	"github.com/KyleMoser/cosmos-client/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestIbcConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Get the chain RPC URI from the Osmosis mainnet chain registry
	chainClientConfigOsmosis, err := client.GetChain(context.Background(), "osmosis", logger)
	if err != nil {
		t.Fail()
	}

	osmosisChainClient, err := client.NewChainClient(logger, chainClientConfigOsmosis, "/home/kyle/.osmosisd", nil, nil)
	if err != nil {
		t.Fail()
	}

	cosmosHubIbc, err := osmosisChainClient.GetIbcConfig("cosmoshub")
	if err != nil {
		t.Fail()
	}

	cosmosHubChannelID := cosmosHubIbc.Channels[0].Chain1.ChannelId
	require.Equal(t, cosmosHubChannelID, "channel-0")
}
