package transactions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elrond-go-core/core"
)

// SerializeTokens will serialize the provided tokens data in a way that Elastic Search expects a bulk request
func (tdp *txsDatabaseProcessor) SerializeTokens(tokens []*data.TokenInfo) ([]*bytes.Buffer, error) {
	buffSlice := data.NewBufferSlice()
	for _, tokenData := range tokens {
		meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s" } }%s`, tokenData.Token, "\n"))
		serializedData, errMarshal := json.Marshal(tokenData)
		if errMarshal != nil {
			return nil, errMarshal
		}

		err := buffSlice.PutData(meta, serializedData)
		if err != nil {
			return nil, err
		}
	}

	return buffSlice.Buffers(), nil
}

// SerializeScResults will serialize the provided smart contract results in a way that Elastic Search expects a bulk request
func (tdp *txsDatabaseProcessor) SerializeScResults(scResults []*data.ScResult) ([]*bytes.Buffer, error) {
	buffSlice := data.NewBufferSlice()
	for _, sc := range scResults {
		meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s" } }%s`, sc.Hash, "\n"))
		serializedData, errPrepareSc := json.Marshal(sc)
		if errPrepareSc != nil {
			return nil, errPrepareSc
		}

		err := buffSlice.PutData(meta, serializedData)
		if err != nil {
			return nil, err
		}
	}

	return buffSlice.Buffers(), nil
}

// SerializeReceipts will serialize the receipts in a way that Elastic Search expects a bulk request
func (tdp *txsDatabaseProcessor) SerializeReceipts(receipts []*data.Receipt) ([]*bytes.Buffer, error) {
	buffSlice := data.NewBufferSlice()
	for _, rec := range receipts {
		meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s" } }%s`, rec.Hash, "\n"))
		serializedData, errPrepareReceipt := json.Marshal(rec)
		if errPrepareReceipt != nil {
			return nil, errPrepareReceipt
		}

		err := buffSlice.PutData(meta, serializedData)
		if err != nil {
			return nil, err
		}
	}

	return buffSlice.Buffers(), nil
}

// SerializeTransactions will serialize the transactions in a way that Elastic Search expects a bulk request
func (tdp *txsDatabaseProcessor) SerializeTransactions(
	transactions []*data.Transaction,
	txHashStatus map[string]string,
	selfShardID uint32,
) ([]*bytes.Buffer, error) {
	buffSlice := data.NewBufferSlice()
	for _, tx := range transactions {
		meta, serializedData, err := prepareSerializedDataForATransaction(tx, selfShardID)
		if err != nil {
			return nil, err
		}

		err = buffSlice.PutData(meta, serializedData)
		if err != nil {
			return nil, err
		}
	}

	for txHash, status := range txHashStatus {
		metaData := []byte(fmt.Sprintf(`{"update":{"_id":"%s", "_type": "_doc"}}%s`, txHash, "\n"))

		newTx := &data.Transaction{
			Status: status,
		}
		marshaledTx, err := json.Marshal(newTx)
		if err != nil {
			return nil, err
		}

		serializedData := []byte(fmt.Sprintf(`{"script": {"source": "ctx._source.status = params.status","lang": "painless","params": {"status": "%s"}},"upsert": %s }`, status, string(marshaledTx)))
		err = buffSlice.PutData(metaData, serializedData)
		if err != nil {
			return nil, err
		}
	}

	return buffSlice.Buffers(), nil
}

func prepareSerializedDataForATransaction(
	tx *data.Transaction,
	selfShardID uint32,
) ([]byte, []byte, error) {
	metaData := []byte(fmt.Sprintf(`{"update":{"_id":"%s", "_type": "_doc"}}%s`, tx.Hash, "\n"))
	marshaledTx, err := json.Marshal(tx)
	if err != nil {
		return nil, nil, err
	}

	if isCrossShardOnSourceShard(tx, selfShardID) {
		// if transaction is cross-shard and current shard ID is source, use upsert without updating anything
		serializedData :=
			[]byte(fmt.Sprintf(`{"script":{"source":"return"},"upsert":%s}`,
				string(marshaledTx)))
		log.Trace("indexer tx is on sender shard", "metaData", string(metaData), "serializedData", string(serializedData))

		return metaData, serializedData, nil
	}

	if isNFTTransferOrMultiTransfer(tx) {
		serializedData, errPrep := prepareNFTESDTTransferOrMultiESDTTransfer(marshaledTx)
		if errPrep != nil {
			return nil, nil, err
		}

		return metaData, serializedData, nil
	}

	// transaction is intra-shard, invalid or cross-shard destination me
	meta := []byte(fmt.Sprintf(`{ "index" : { "_id" : "%s", "_type" : "%s" } }%s`, tx.Hash, "_doc", "\n"))
	log.Trace("indexer tx is intra shard or invalid tx", "meta", string(meta), "marshaledTx", string(marshaledTx))

	return meta, marshaledTx, nil
}

func prepareNFTESDTTransferOrMultiESDTTransfer(marshaledTx []byte) ([]byte, error) {
	serializedData := []byte(fmt.Sprintf(`{"script":{"source":"`+
		`def status = ctx._source.status;`+
		`ctx._source = params.tx;`+
		`ctx._source.status = status;`+
		`","lang": "painless","params":`+
		`{"tx": %s}},"upsert":%s}`,
		string(marshaledTx), string(marshaledTx)))

	return serializedData, nil
}

func isNFTTransferOrMultiTransfer(tx *data.Transaction) bool {
	if len(tx.SmartContractResults) < 0 || tx.SenderShard != tx.ReceiverShard {
		return false
	}

	splitData := strings.Split(string(tx.Data), atSeparator)
	if len(splitData) < 4 {
		return false
	}

	return splitData[0] == core.BuiltInFunctionESDTNFTTransfer || splitData[0] == core.BuiltInFunctionMultiESDTNFTTransfer
}
