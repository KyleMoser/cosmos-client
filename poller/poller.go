package poller

import (
	"errors"
	"math/rand"
	"sync"

	"github.com/KyleMoser/cosmos-client/client"
)

type Querier struct {
	blockObservers           map[string]BlockHeightObserver   // chainID is the map key. observers can choose to listen for new blocks on particular chains.
	chainClients             map[string][]*client.ChainClient // chainID is the map key. multiple chain clients used for liveness.
	activeBlockHeightQueries map[string]*sync.Mutex
}

func (q *Querier) Subscribe(o BlockHeightObserver, chainID string) (bool, error) {
	return false, nil
}
func (q *Querier) Unsubscribe(o BlockHeightObserver, chainID string) (bool, error) {
	return false, nil
}

func (q *Querier) getLatestBlocks(chain string) {
	//Prevent multiple concurrent queries to the same chain
	q.activeBlockHeightQueries[chain].Lock()
	defer q.activeBlockHeightQueries[chain].Unlock()

	chainClient, err := q.randLiveChain(chain)
	if err == nil {
		newBlocks := make(chan int64)
		AwaitBlocks(chainClient.Config.RPCAddr, newBlocks)
		lastHeight := int64(0)
		for {
			newHeight := <-newBlocks
			if newHeight > lastHeight {
				lastHeight = newHeight
				q.Notify(chain, newHeight)
			}
		}
	}
}

func (q *Querier) awaitBlocks() {
	for {
		observedChains := q.getObservedChains()
		for _, chain := range observedChains {
			if _, ok := q.activeBlockHeightQueries[chain]; !ok {
				q.activeBlockHeightQueries[chain] = new(sync.Mutex)
			}
			go q.getLatestBlocks(chain)
		}
	}
}

func (q *Querier) randLiveChain(chain string) (*client.ChainClient, error) {
	chains := q.chainClients[chain]

	liveChains := []*client.ChainClient{}
	for _, chain := range chains {
		if chain.IsActive() {
			liveChains = append(liveChains, chain)
		}
	}

	if len(liveChains) == 0 {
		return nil, errors.New("No active chains")
	} else if len(liveChains) == 1 {
		return liveChains[0], nil
	}

	return liveChains[rand.Intn(len(liveChains))], nil
}

// Not all chains will have observers and it's pointless to query chains if nobody wants the results.
func (q *Querier) getObservedChains() []string {
	observedChains := make([]string, len(q.blockObservers))

	i := 0
	for k := range q.blockObservers {
		observedChains[i] = k
		i++
	}

	return observedChains
}

func (q *Querier) Notify(chain string, height int64) {

}

// type blockPoller[T any] struct {
// 	pollFunc     func(ctx context.Context, height uint64) (T, error)
// 	chainClients []*client.ChainClient
// }

// func NewBlockPoller(chainClients ...*client.ChainClient) {

// }

// func (p blockPoller[T]) DoPoll(ctx context.Context, startHeight, maxHeight uint64) (T, error) {
// 	if maxHeight < startHeight {
// 		panic("maxHeight must be greater than or equal to startHeight")
// 	}

// 	var (
// 		pollErr error
// 		zero    T
// 	)

// 	cursor := startHeight
// 	for cursor <= maxHeight {
// 		curHeight, err := p.CurrentHeight(ctx)
// 		if err != nil {
// 			return zero, err
// 		}
// 		if cursor > curHeight {
// 			continue
// 		}

// 		found, findErr := p.PollFunc(ctx, cursor)

// 		if findErr != nil {
// 			pollErr = findErr
// 			cursor++
// 			continue
// 		}

// 		return found, nil
// 	}
// 	return zero, pollErr
// }
