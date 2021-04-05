package transactions

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elrond-go/core"
	nodeData "github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/receipt"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/sharding"
)

type dbTransactionBuilder struct {
	esdtProc               *esdtTransactionProcessor
	addressPubkeyConverter core.PubkeyConverter
	shardCoordinator       sharding.Coordinator
	txFeeCalculator        process.TransactionFeeCalculator
}

func newTransactionDBBuilder(
	addressPubkeyConverter core.PubkeyConverter,
	shardCoordinator sharding.Coordinator,
	txFeeCalculator process.TransactionFeeCalculator,
) *dbTransactionBuilder {
	esdtProc := newEsdtTransactionHandler()

	return &dbTransactionBuilder{
		esdtProc:               esdtProc,
		addressPubkeyConverter: addressPubkeyConverter,
		shardCoordinator:       shardCoordinator,
		txFeeCalculator:        txFeeCalculator,
	}
}

func (dtb *dbTransactionBuilder) prepareTransaction(
	tx *transaction.Transaction,
	txHash []byte,
	mbHash []byte,
	mb *block.MiniBlock,
	header nodeData.HeaderHandler,
	txStatus string,
) *data.Transaction {
	var tokenIdentifier, esdtValue string
	isESDTTx := dtb.esdtProc.isESDTTx(tx)
	if isESDTTx {
		tokenIdentifier, esdtValue = dtb.esdtProc.getTokenIdentifierAndValue(tx)
	}

	gasUsed := dtb.txFeeCalculator.ComputeGasLimit(tx)
	fee := dtb.txFeeCalculator.ComputeTxFeeBasedOnGasUsed(tx, gasUsed)

	return &data.Transaction{
		Hash:                 hex.EncodeToString(txHash),
		MBHash:               hex.EncodeToString(mbHash),
		Nonce:                tx.Nonce,
		Round:                header.GetRound(),
		Value:                tx.Value.String(),
		Receiver:             dtb.addressPubkeyConverter.Encode(tx.RcvAddr),
		Sender:               dtb.addressPubkeyConverter.Encode(tx.SndAddr),
		ReceiverShard:        mb.ReceiverShardID,
		SenderShard:          mb.SenderShardID,
		GasPrice:             tx.GasPrice,
		GasLimit:             tx.GasLimit,
		Data:                 tx.Data,
		Signature:            hex.EncodeToString(tx.Signature),
		Timestamp:            time.Duration(header.GetTimeStamp()),
		Status:               txStatus,
		EsdtTokenIdentifier:  tokenIdentifier,
		EsdtValue:            esdtValue,
		GasUsed:              gasUsed,
		Fee:                  fee.String(),
		ReceiverUserName:     tx.RcvUserName,
		SenderUserName:       tx.SndUserName,
		ReceiverAddressBytes: tx.RcvAddr,
	}
}

func (dtb *dbTransactionBuilder) prepareRewardTransaction(
	rTx *rewardTx.RewardTx,
	txHash []byte,
	mbHash []byte,
	mb *block.MiniBlock,
	header nodeData.HeaderHandler,
	txStatus string,
) *data.Transaction {
	return &data.Transaction{
		Hash:          hex.EncodeToString(txHash),
		MBHash:        hex.EncodeToString(mbHash),
		Nonce:         0,
		Round:         rTx.Round,
		Value:         rTx.Value.String(),
		Receiver:      dtb.addressPubkeyConverter.Encode(rTx.RcvAddr),
		Sender:        fmt.Sprintf("%d", core.MetachainShardId),
		ReceiverShard: mb.ReceiverShardID,
		SenderShard:   mb.SenderShardID,
		GasPrice:      0,
		GasLimit:      0,
		Data:          make([]byte, 0),
		Signature:     "",
		Timestamp:     time.Duration(header.GetTimeStamp()),
		Status:        txStatus,
	}
}

func (dtb *dbTransactionBuilder) prepareSmartContractResult(
	scHash string,
	sc *smartContractResult.SmartContractResult,
	header nodeData.HeaderHandler,
) *data.ScResult {
	relayerAddr := ""
	if len(sc.RelayerAddr) > 0 {
		relayerAddr = dtb.addressPubkeyConverter.Encode(sc.RelayerAddr)
	}

	var tokenIdentifier, esdtValue string

	isESDTTx := dtb.esdtProc.isESDTTx(sc)
	if isESDTTx {
		tokenIdentifier, esdtValue = dtb.esdtProc.getTokenIdentifierAndValue(sc)
	}

	return &data.ScResult{
		Hash:                hex.EncodeToString([]byte(scHash)),
		Nonce:               sc.Nonce,
		GasLimit:            sc.GasLimit,
		GasPrice:            sc.GasPrice,
		Value:               sc.Value.String(),
		Sender:              dtb.addressPubkeyConverter.Encode(sc.SndAddr),
		Receiver:            dtb.addressPubkeyConverter.Encode(sc.RcvAddr),
		RelayerAddr:         relayerAddr,
		RelayedValue:        sc.RelayedValue.String(),
		Code:                string(sc.Code),
		Data:                sc.Data,
		PrevTxHash:          hex.EncodeToString(sc.PrevTxHash),
		OriginalTxHash:      hex.EncodeToString(sc.OriginalTxHash),
		CallType:            strconv.Itoa(int(sc.CallType)),
		CodeMetadata:        sc.CodeMetadata,
		ReturnMessage:       string(sc.ReturnMessage),
		EsdtTokenIdentifier: tokenIdentifier,
		EsdtValue:           esdtValue,
		Timestamp:           time.Duration(header.GetTimeStamp()),
	}
}

func (dtb *dbTransactionBuilder) prepareReceipt(
	recHash string,
	rec *receipt.Receipt,
	header nodeData.HeaderHandler,
) *data.Receipt {
	return &data.Receipt{
		Hash:      hex.EncodeToString([]byte(recHash)),
		Value:     rec.Value.String(),
		Sender:    dtb.addressPubkeyConverter.Encode(rec.SndAddr),
		Data:      string(rec.Data),
		TxHash:    hex.EncodeToString(rec.TxHash),
		Timestamp: time.Duration(header.GetTimeStamp()),
	}
}

func (dtb *dbTransactionBuilder) addScrsReceiverToAlteredAccounts(
	alteredAddress map[string]*data.AlteredAccount,
	scrs []*data.ScResult,
) {
	for _, scr := range scrs {
		receiverAddr, _ := dtb.addressPubkeyConverter.Decode(scr.Receiver)
		shardID := dtb.shardCoordinator.ComputeId(receiverAddr)
		if shardID != dtb.shardCoordinator.SelfId() {
			continue
		}

		egldBalanceNotChanged := scr.Value == "" || scr.Value == "0"
		esdtBalanceNotChanged := scr.EsdtValue == "" || scr.EsdtValue == "0"
		if egldBalanceNotChanged && esdtBalanceNotChanged {
			// the smart contract results that don't alter the balance of the receiver address should be ignored
			continue
		}
		encodedReceiverAddress := scr.Receiver
		alteredAddress[encodedReceiverAddress] = &data.AlteredAccount{
			IsESDTOperation: scr.EsdtTokenIdentifier != "" && scr.EsdtValue != "",
			TokenIdentifier: scr.EsdtTokenIdentifier,
		}
	}
}
