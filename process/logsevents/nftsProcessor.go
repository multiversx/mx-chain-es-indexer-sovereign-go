package logsevents

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ElrondNetwork/elastic-indexer-go/converters"
	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elastic-indexer-go/process/tags"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/core/check"
	nodeData "github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/ElrondNetwork/elrond-go/sharding"
	"github.com/ElrondNetwork/elrond-vm-common/data/esdt"
)

type nftsProcessor struct {
	pubKeyConverter          core.PubkeyConverter
	nftOperationsIdentifiers map[string]struct{}
	shardCoordinator         sharding.Coordinator
	marshalizer              marshal.Marshalizer
}

func newNFTsProcessor(
	shardCoordinator sharding.Coordinator,
	pubKeyConverter core.PubkeyConverter,
	marshalizer marshal.Marshalizer,
) *nftsProcessor {
	return &nftsProcessor{
		shardCoordinator: shardCoordinator,
		pubKeyConverter:  pubKeyConverter,
		marshalizer:      marshalizer,
		nftOperationsIdentifiers: map[string]struct{}{
			core.BuiltInFunctionESDTNFTTransfer:    {},
			core.BuiltInFunctionESDTNFTBurn:        {},
			core.BuiltInFunctionESDTNFTAddQuantity: {},
			core.BuiltInFunctionESDTNFTCreate:      {},
		},
	}
}

func (np *nftsProcessor) processLogAndEventsNFTs(
	logsAndEvents map[string]nodeData.LogHandler,
	accounts data.AlteredAccountsHandler,
	timestamp uint64,
	txsMap map[string]*data.Transaction,
	scrsMap map[string]*data.ScResult,
) (data.TokensHandler, tags.CountTags) {
	tagsCount := tags.NewTagsCount()
	tokens := data.NewTokensInfo()

	if logsAndEvents == nil || accounts == nil {
		return tokens, tagsCount
	}

	for logHash, txLog := range logsAndEvents {
		if check.IfNil(txLog) {
			continue
		}

		np.processNFTOperationLog(logHash, txLog, accounts, tokens, timestamp, tagsCount, txsMap, scrsMap)
	}

	return tokens, tagsCount
}

func (np *nftsProcessor) processNFTOperationLog(
	logHash string,
	txLog nodeData.LogHandler,
	accounts data.AlteredAccountsHandler,
	tokens data.TokensHandler,
	timestamp uint64,
	tagsCount tags.CountTags,
	txsMap map[string]*data.Transaction,
	scrsMap map[string]*data.ScResult,
) {
	events := txLog.GetLogEvents()
	if len(events) == 0 {
		return
	}

	logHashHexEncoded := hex.EncodeToString([]byte(logHash))
	for _, event := range events {
		tokenIdentifier := np.processEvent(event, accounts, tokens, timestamp, tagsCount)

		tx, ok := txsMap[logHashHexEncoded]
		if ok {
			tx.EsdtTokenIdentifier = tokenIdentifier
			continue
		}

		scr, ok := scrsMap[logHashHexEncoded]
		if ok {
			scr.EsdtTokenIdentifier = tokenIdentifier
			continue
		}
	}
}

func (np *nftsProcessor) processEvent(
	event nodeData.EventHandler,
	accounts data.AlteredAccountsHandler,
	tokens data.TokensHandler,
	timestamp uint64,
	tagsCount tags.CountTags,
) string {
	_, ok := np.nftOperationsIdentifiers[string(event.GetIdentifier())]
	if !ok {
		return ""
	}
	sender := event.GetAddress()

	if np.shardCoordinator.ComputeId(sender) == np.shardCoordinator.SelfId() {
		np.processNFTEventOnSender(event, accounts, tokens, timestamp, tagsCount)
	}

	// topics contains:
	// [0] -- token identifier
	// [1] -- nonce of the NFT (bytes)
	// [2] -- receiver NFT address -- in case of NFTTransfer OR ESDT token data in case of NFTCreate
	topics := event.GetTopics()
	token := string(topics[0])
	nonceBig := big.NewInt(0).SetBytes(topics[1])
	identifier := converters.ComputeTokenIdentifier(token, nonceBig.Uint64())
	if string(event.GetIdentifier()) != core.BuiltInFunctionESDTNFTTransfer || len(topics) < 3 {
		return identifier
	}

	receiver := topics[2]
	if np.shardCoordinator.ComputeId(receiver) != np.shardCoordinator.SelfId() {
		return identifier
	}

	encodedReceiver := np.pubKeyConverter.Encode(receiver)
	accounts.Add(encodedReceiver, &data.AlteredAccount{
		IsNFTOperation:  true,
		TokenIdentifier: token,
		NFTNonce:        nonceBig.Uint64(),
	})

	return identifier
}

func (np *nftsProcessor) processNFTEventOnSender(
	event nodeData.EventHandler,
	accounts data.AlteredAccountsHandler,
	tokensCreateInfo data.TokensHandler,
	timestamp uint64,
	tagsCount tags.CountTags,
) {
	sender := event.GetAddress()
	topics := event.GetTopics()
	token := string(topics[0])
	nonceBig := big.NewInt(0).SetBytes(topics[1])
	bech32Addr := np.pubKeyConverter.Encode(sender)

	alteredAccount := &data.AlteredAccount{
		IsNFTOperation:  true,
		TokenIdentifier: token,
		NFTNonce:        nonceBig.Uint64(),
	}

	accounts.Add(bech32Addr, alteredAccount)

	shouldReturn := string(event.GetIdentifier()) != core.BuiltInFunctionESDTNFTCreate || len(topics) < 3
	if shouldReturn {
		return
	}

	esdtTokenBytes := topics[2]
	esdtToken := &esdt.ESDigitalToken{}
	err := np.marshalizer.Unmarshal(esdtToken, esdtTokenBytes)
	if err != nil {
		return
	}

	tokenMetaData := converters.PrepareTokenMetaData(np.pubKeyConverter, esdtToken)
	tokensCreateInfo.Add(&data.TokenInfo{
		Token:      token,
		Identifier: converters.ComputeTokenIdentifier(token, nonceBig.Uint64()),
		Timestamp:  time.Duration(timestamp),
		Data:       tokenMetaData,
	})

	if tokenMetaData != nil {
		tagsCount.ParseTags(tokenMetaData.Tags)
	}
}
