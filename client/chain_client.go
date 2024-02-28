package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	registry "github.com/strangelove-ventures/cosmos-client/client/chain_registry"
	"github.com/strangelove-ventures/cosmos-client/client/rpc"

	provtypes "github.com/cometbft/cometbft/light/provider"
	prov "github.com/cometbft/cometbft/light/provider/http"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/gogoproto/proto"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	// Variables used for retries
	RtyAttNum = uint(5)
	RtyAtt    = retry.Attempts(RtyAttNum)
	RtyDel    = retry.Delay(time.Millisecond * 400)
	RtyErr    = retry.LastErrorOnly(true)
)

type ChainClient struct {
	log *zap.Logger

	Config         *ChainClientConfig
	Keybase        keyring.Keyring
	KeyringOptions []keyring.Option
	RPCClient      rpcclient.Client
	LightProvider  provtypes.Provider
	Input          io.Reader
	Output         io.Writer
	// TODO: GRPC Client type?
	rpcLiveness
	Codec Codec
}

type rpcLiveness struct {
	isActive    bool
	lastChecked time.Time
	lastActive  time.Time
}

func (cc *ChainClient) IsActive() bool {
	return cc.rpcLiveness.isActive
}

// CliContext creates a new Cosmos SDK client context
func (cc *ChainClient) CliContext() client.Context {
	return client.Context{
		Client:            cc.RPCClient,
		ChainID:           cc.Config.ChainID,
		InterfaceRegistry: cc.Codec.InterfaceRegistry,
		Input:             os.Stdin,
		Output:            os.Stdout,
		OutputFormat:      "json",
		LegacyAmino:       cc.Codec.Amino,
		TxConfig:          cc.Codec.TxConfig,
	}
}

// HealthChecks will continuously check the liveness of each of the given ChainClients.
// If a ChainClient's RPC node is active, chainClient.IsActive() will return true.
func HealthChecks(chainClients ...*ChainClient) {
	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				//Check all of the ChainClients to see if they respond to RPC 'get height' queries.
				var wg sync.WaitGroup
				for _, cc := range chainClients {
					cc := cc
					if time.Since(cc.rpcLiveness.lastChecked) > (9 * time.Second) {
						wg.Add(1)

						go func() {
							defer wg.Done()
							cc.healthCheck()
						}()
					}
				}

				wg.Wait()

				//Stop the liveness checks if none of the RPC clients have been active for at least 10 minutes
				lastActiveTime := time.Now().Add(time.Duration(-10) * time.Minute)
				for _, cc := range chainClients {
					if cc.rpcLiveness.isActive {
						break
					}

					if cc.rpcLiveness.lastActive.After(lastActiveTime) {
						break
					}
				}

				close(quit)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *ChainClient) GetIbcTransferConfig(destChain string) (srcChannel, srcPort, clientId string, err error) {
	ibcConfig, err := c.GetIbcConfig(destChain)
	if err != nil {
		return "", "", "", err
	} else if len(ibcConfig.Channels) == 0 {
		return "", "", "", errors.New("unexpected chain configuration 'channels' not found")
	}

	clientId = ibcConfig.Chain1.ClientId
	srcChannel = ibcConfig.Channels[0].Chain1.ChannelId
	srcPort = ibcConfig.Channels[0].Chain1.PortId
	return
}

// Get the IBC configuration where this chain is the source and destChain is the IBC endpoint.
func (c *ChainClient) GetIbcConfig(destChain string) (registry.IbcConfig, error) {
	// A chain registry entry could either be under e.g. https://github.com/cosmos/chain-registry/blob/master/_IBC/juno-osmosis.json
	// OR it could be under e.g. https://github.com/cosmos/chain-registry/blob/master/_IBC/osmosis-juno.json
	chainRegURL := fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/_IBC/%s-%s.json", c.Config.ChainName, destChain)
	chainRegURLReversed := fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/_IBC/%s-%s.json", destChain, c.Config.ChainName)
	reversed := false

	res, err := http.Get(chainRegURL)
	if err != nil {
		return registry.IbcConfig{}, err
	}

	if res.StatusCode == 404 {
		reversed = true
		res, err = http.Get(chainRegURLReversed)
		if err != nil {
			return registry.IbcConfig{}, err
		} else if res.StatusCode != 200 {
			return registry.IbcConfig{}, fmt.Errorf("expected http 200, got %d", res.StatusCode)
		}
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return registry.IbcConfig{}, fmt.Errorf("IBC config not found: response code: %d: GET failed: %s", res.StatusCode, chainRegURL)
	}
	if res.StatusCode != http.StatusOK {
		return registry.IbcConfig{}, fmt.Errorf("response code: %d: GET failed: %s", res.StatusCode, chainRegURL)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return registry.IbcConfig{}, err
	}

	var conf registry.IbcConfig
	if err := json.Unmarshal([]byte(body), &conf); err != nil {
		return registry.IbcConfig{}, err
	}

	// In the registry file contents, we must flip all references to chain1 and chain2. See example URLs above.
	if reversed {
		tmp := conf.Chain1
		conf.Chain1 = conf.Chain2
		conf.Chain2 = tmp

		for i := range conf.Channels {
			tmp := conf.Channels[i].Chain1
			conf.Channels[i].Chain1 = conf.Channels[i].Chain2
			conf.Channels[i].Chain2 = tmp
		}
	}
	return conf, nil
}

func (cc *ChainClient) healthCheck() bool {
	_, err := cc.QueryLatestHeight(context.Background())
	cc.rpcLiveness.isActive = err == nil
	cc.rpcLiveness.lastChecked = time.Now()
	if cc.rpcLiveness.isActive {
		cc.rpcLiveness.lastActive = time.Now()
	}
	return cc.rpcLiveness.isActive
}

// Chain client where keys are in 'rootKeyDirectory/keyring-test' (or whichever keyring-backend is chosen)
func NewChainClientWithRootKeyDir(log *zap.Logger, ccc *ChainClientConfig, rootKeyDirectory string, input io.Reader, output io.Writer, kro ...keyring.Option) (*ChainClient, error) {
	ccc.KeyDirectory = rootKeyDirectory
	cc := &ChainClient{
		log: log,

		KeyringOptions: kro,
		Config:         ccc,
		Input:          input,
		Output:         output,
		Codec:          MakeCodec(ccc.Modules, ccc.ExtraCodecs, ccc.AccountPrefix, ccc.AccountPrefix+"valoper"),
	}
	if err := cc.Init(); err != nil {
		return nil, err
	}
	return cc, nil
}

func NewChainClient(log *zap.Logger, ccc *ChainClientConfig, homepath string, input io.Reader, output io.Writer, kro ...keyring.Option) (*ChainClient, error) {
	ccc.KeyDirectory = keysDir(homepath, ccc.ChainID)
	cc := &ChainClient{
		log: log,

		KeyringOptions: kro,
		Config:         ccc,
		Input:          input,
		Output:         output,
		Codec:          MakeCodec(ccc.Modules, ccc.ExtraCodecs, ccc.AccountPrefix, ccc.AccountPrefix+"valoper"),
	}
	if err := cc.Init(); err != nil {
		return nil, err
	}
	return cc, nil
}

func (cc *ChainClient) Init() error {
	// TODO: test key directory and return error if not created
	keybase, err := keyring.New(cc.Config.ChainID, cc.Config.KeyringBackend, cc.Config.KeyDirectory, cc.Input, cc.Codec.Marshaler, cc.KeyringOptions...)
	if err != nil {
		return err
	}
	// TODO: figure out how to deal with input or maybe just make all keyring backends test?

	timeout, _ := time.ParseDuration(cc.Config.Timeout)
	rpcClient, err := rpc.NewRPCClient(cc.Config.RPCAddr, timeout)
	if err != nil {
		return err
	}

	lightprovider, err := prov.New(cc.Config.ChainID, cc.Config.RPCAddr)
	if err != nil {
		return err
	}

	cc.RPCClient = rpcClient
	cc.LightProvider = lightprovider
	cc.Keybase = keybase

	return nil
}

func (cc *ChainClient) GetKeyAddress() (sdk.AccAddress, error) {
	info, err := cc.Keybase.Key(cc.Config.Key)
	if err != nil {
		return nil, err
	}
	return info.GetAddress()
}

// AccountFromKeyOrAddress returns an account from either a key or an address
// if empty string is passed in this returns the default key's address
func (cc *ChainClient) AccountFromKeyOrAddress(keyOrAddress string) (out sdk.AccAddress, err error) {
	switch {
	case keyOrAddress == "":
		out, err = cc.GetKeyAddress()
	case cc.KeyExists(keyOrAddress):
		cc.Config.Key = keyOrAddress
		out, err = cc.GetKeyAddress()
	default:
		out, err = cc.DecodeBech32AccAddr(keyOrAddress)
	}
	return
}

func keysDir(home, chainID string) string {
	return path.Join(home, "keys", chainID)
}

// TODO: actually do something different here have a couple of levels of verbosity
func (cc *ChainClient) PrintTxResponse(res *sdk.TxResponse) error {
	return cc.PrintObject(res)
}

func (cc *ChainClient) HandleAndPrintMsgSend(res *sdk.TxResponse, err error) error {
	if err != nil {
		if res != nil {
			return fmt.Errorf("failed to withdraw rewards: code(%d) msg(%s)", res.Code, res.Logs)
		}
		return fmt.Errorf("failed to withdraw rewards: err(%w)", err)
	}
	return cc.PrintTxResponse(res)
}

func (cc *ChainClient) PrintObject(res interface{}) error {
	var (
		bz  []byte
		err error
	)
	switch cc.Config.OutputFormat {
	case "json":
		if m, ok := res.(proto.Message); ok {
			bz, err = cc.MarshalProto(m)
		} else {
			bz, err = json.Marshal(res)
		}
		if err != nil {
			return err
		}
	case "indent":
		if m, ok := res.(proto.Message); ok {
			bz, err = cc.MarshalProto(m)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer([]byte{})
			if err = json.Indent(buf, bz, "", "  "); err != nil {
				return err
			}
			bz = buf.Bytes()
		} else {
			bz, err = json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
		}
	case "yaml":
		bz, err = yaml.Marshal(res)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown output type: %s", cc.Config.OutputFormat)
	}
	fmt.Fprint(cc.Output, string(bz), "\n")
	return nil
}

func (cc *ChainClient) MarshalProto(res proto.Message) ([]byte, error) {
	return cc.Codec.Marshaler.MarshalJSON(res)
}
