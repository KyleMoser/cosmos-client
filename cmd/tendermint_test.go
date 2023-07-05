package cmd_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/KyleMoser/cosmos-client/cmd"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/rpc/client/mocks"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// OverrideClients sets the client override mapping for the chain with the given name.
// This override applies to all subsequent command invocations for this System.
func (s *System) OverrideClients(name string, o cmd.ClientOverrides) {
	if s.clientOverrides == nil {
		s.clientOverrides = map[string]cmd.ClientOverrides{}
	}
	s.clientOverrides[name] = o
}

// System is a system under test.
type System struct {
	HomeDir string

	clientOverrides map[string]cmd.ClientOverrides
}

// NewSystem creates a new system with a home dir associated with a temp dir belonging to t.
//
// The returned System does not store a reference to t;
// some of its methods expect a *testing.T as an argument.
// This allows creating one instance of System to be shared with subtests.
func NewSystem(t *testing.T) *System {
	t.Helper()

	homeDir := t.TempDir()

	return &System{
		HomeDir: homeDir,
	}
}

// MustRun calls Run, but also calls t.Fatal if RunResult.Err is not nil.
func (s *System) MustRun(t *testing.T, args ...string) RunResult {
	t.Helper()

	return s.MustRunWithInput(t, bytes.NewReader(nil), args...)
}

// Run calls s.RunWithInput with an empty stdin.
func (s *System) Run(log *zap.Logger, args ...string) RunResult {
	return s.RunWithInput(log, bytes.NewReader(nil), args...)
}

// RunResult is the stdout and stderr resulting from a call to (*System).Run,
// and any error that was returned.
type RunResult struct {
	Stdout, Stderr bytes.Buffer

	Err error
}

// RunWithInput executes the root command with the given args,
// providing in as the command's standard input,
// and returns a RunResult that has its Stdout and Stderr populated.
func (s *System) RunWithInput(log *zap.Logger, in io.Reader, args ...string) RunResult {
	rootCmd := cmd.NewRootCmd(log, zap.NewAtomicLevel(), s.clientOverrides)
	rootCmd.SetIn(in)
	// cmd.Execute also sets SilenceUsage,
	// so match that here for more correct assertions.
	rootCmd.SilenceUsage = true

	var res RunResult
	rootCmd.SetOutput(&res.Stdout)
	rootCmd.SetErr(&res.Stderr)

	// Prepend the system's home directory to any provided args.
	args = append([]string{"--home", s.HomeDir}, args...)
	rootCmd.SetArgs(args)

	res.Err = rootCmd.Execute()
	return res
}

// MustRunWithInput calls RunWithInput, but also calls t.Fatal if RunResult.Err is not nil.
func (s *System) MustRunWithInput(t *testing.T, in io.Reader, args ...string) RunResult {
	t.Helper()

	res := s.RunWithInput(zaptest.NewLogger(t), in, args...)
	if res.Err != nil {
		t.Logf("Error executing %v: %v", args, res.Err)
		t.Logf("Stdout: %q", res.Stdout.String())
		t.Logf("Stderr: %q", res.Stderr.String())
		t.FailNow()
	}

	return res
}

func TestTendermintStatus(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	// Arbitrary status response with a few fields filled in.
	mockStatus := coretypes.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			Moniker: "foo bar",
		},
		SyncInfo: coretypes.SyncInfo{
			LatestBlockHeight: 123,
		},
		ValidatorInfo: coretypes.ValidatorInfo{
			VotingPower: 5,
		},
	}
	mc := new(mocks.Client)
	mc.On("Status", mock.Anything).Return(&mockStatus, nil)

	sys.OverrideClients("cosmoshub", cmd.ClientOverrides{
		RPCClient: mc,
	})

	// tm status prints the received status as JSON.
	// Nothing should output on stderr.
	res := sys.MustRun(t, "tendermint", "status")
	require.Empty(t, res.Stderr.String())

	var gotStatus coretypes.ResultStatus
	require.NoError(t, json.Unmarshal(res.Stdout.Bytes(), &gotStatus))

	require.Empty(t, cmp.Diff(mockStatus, gotStatus, cmpopts.EquateEmpty()))
}

// A fixed mnemonic and its resulting cosmos address, helpful for tests that need a mnemonic.
const (
	ZeroMnemonic   = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art"
	ZeroCosmosAddr = "cosmos1r5v5srda7xfth3hn2s26txvrcrntldjumt8mhl"
)
