package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	provtypes "github.com/cometbft/cometbft/light/provider"
	prov "github.com/cometbft/cometbft/light/provider/http"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	libclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
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

func (cc *ChainClient) healthCheck() bool {
	_, err := cc.QueryLatestHeight(context.Background())
	cc.rpcLiveness.isActive = err == nil
	cc.rpcLiveness.lastChecked = time.Now()
	if cc.rpcLiveness.isActive {
		cc.rpcLiveness.lastActive = time.Now()
	}
	return cc.rpcLiveness.isActive
}

func NewChainClient(log *zap.Logger, ccc *ChainClientConfig, homepath string, input io.Reader, output io.Writer, kro ...keyring.Option) (*ChainClient, error) {
	ccc.KeyDirectory = keysDir(homepath, ccc.ChainID)
	cc := &ChainClient{
		log: log,

		KeyringOptions: kro,
		Config:         ccc,
		Input:          input,
		Output:         output,
		Codec:          MakeCodec(ccc.Modules, ccc.ExtraCodecs),
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
	rpcClient, err := NewRPCClient(cc.Config.RPCAddr, timeout)
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

func NewRPCClient(addr string, timeout time.Duration) (*rpchttp.HTTP, error) {
	httpClient, err := libclient.DefaultHTTPClient(addr)
	if err != nil {
		return nil, err
	}
	httpClient.Timeout = timeout
	rpcClient, err := rpchttp.NewWithClient(addr, "/websocket", httpClient)
	if err != nil {
		return nil, err
	}
	return rpcClient, nil
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
