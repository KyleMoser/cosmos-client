package poller

// Observers get notified when a new block is published to the chain.
type BlockHeightNotifier interface {
	Subscribe(o BlockHeightObserver, chainID string) (bool, error)
	Unsubscribe(o BlockHeightObserver, chainID string) (bool, error)
	Notify(chain string, height int64)
}

type BlockHeightObserver interface {
	Get() chan uint64
}
