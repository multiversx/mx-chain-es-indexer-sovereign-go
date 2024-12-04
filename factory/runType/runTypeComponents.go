package runType

import (
	"github.com/multiversx/mx-chain-es-indexer-go/process/elasticproc/transactions"
)

type runTypeComponents struct {
	txHashExtractor transactions.TxHashExtractor
}

// Close does nothing
func (rtc *runTypeComponents) Close() error {
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (rtc *runTypeComponents) IsInterfaceNil() bool {
	return rtc == nil
}
