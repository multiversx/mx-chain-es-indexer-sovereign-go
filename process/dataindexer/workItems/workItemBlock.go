package workItems

import (
	"fmt"
	"time"

	"github.com/multiversx/mx-chain-core-go/core/check"
	"github.com/multiversx/mx-chain-core-go/data/outport"
	logger "github.com/multiversx/mx-chain-logger-go"
)

var log = logger.GetOrCreate("indexer/workItems")

type itemBlock struct {
	indexer                saveBlockIndexer
	outportBlockWithHeader *outport.OutportBlockWithHeader
}

// NewItemBlock will create a new instance of ItemBlock
func NewItemBlock(
	indexer saveBlockIndexer,
	outportBlock *outport.OutportBlockWithHeader,
) WorkItemHandler {
	return &itemBlock{
		indexer:                indexer,
		outportBlockWithHeader: outportBlock,
	}
}

// Save will prepare and save a block item in elasticsearch database
func (wib *itemBlock) Save() error {
	if check.IfNilReflect(wib.outportBlockWithHeader) {
		log.Warn("nil outportBlock block provided when trying to index block, will skip")
		return nil
	}
	if check.IfNil(wib.outportBlockWithHeader.Header) {
		log.Warn("nil header provided when trying to index block, will skip")
		return nil
	}

	defer func(startTime time.Time) {
		log.Debug("wib.SaveBlockData duration", "time", time.Since(startTime))
	}(time.Now())

	log.Debug("indexer: starting indexing block",
		"hash", wib.outportBlockWithHeader.BlockData.HeaderHash,
		"nonce", wib.outportBlockWithHeader.Header.GetNonce())

	if wib.outportBlockWithHeader.TransactionPool == nil {
		wib.outportBlockWithHeader.TransactionPool = &outport.TransactionPool{}
	}

	err := wib.indexer.SaveHeader(wib.outportBlockWithHeader)
	if err != nil {
		return fmt.Errorf("%w when saving header block, hash %s, nonce %d",
			err, wib.outportBlockWithHeader.BlockData.HeaderHash, wib.outportBlockWithHeader.Header.GetNonce())
	}

	if len(wib.outportBlockWithHeader.BlockData.Body.MiniBlocks) == 0 {
		return nil
	}

	err = wib.indexer.SaveMiniblocks(wib.outportBlockWithHeader.Header, wib.outportBlockWithHeader.BlockData.Body)
	if err != nil {
		return fmt.Errorf("%w when saving miniblocks, block hash %s, nonce %d",
			err, wib.outportBlockWithHeader.BlockData.HeaderHash, wib.outportBlockWithHeader.Header.GetNonce())
	}

	err = wib.indexer.SaveTransactions(wib.outportBlockWithHeader)
	if err != nil {
		return fmt.Errorf("%w when saving transactions, block hash %s, nonce %d",
			err, wib.outportBlockWithHeader.BlockData.HeaderHash, wib.outportBlockWithHeader.Header.GetNonce())
	}

	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (wib *itemBlock) IsInterfaceNil() bool {
	return wib == nil
}
