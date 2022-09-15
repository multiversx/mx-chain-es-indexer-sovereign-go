//go:build integrationtests

package integrationtests

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elastic-indexer-go/mock"
	indexerdata "github.com/ElrondNetwork/elastic-indexer-go/process/dataindexer"
	"github.com/ElrondNetwork/elrond-go-core/core"
	coreData "github.com/ElrondNetwork/elrond-go-core/data"
	dataBlock "github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/esdt"
	"github.com/ElrondNetwork/elrond-go-core/data/outport"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/stretchr/testify/require"
)

func TestIndexAccountESDTWithTokenType(t *testing.T) {
	setLogLevelDebug()

	esClient, err := createESClient(esURL)
	require.Nil(t, err)

	// ################ ISSUE NON FUNGIBLE TOKEN ##########################
	shardCoordinator := &mock.ShardCoordinatorMock{
		SelfID: core.MetachainShardId,
	}

	esProc, err := CreateElasticProcessor(esClient, shardCoordinator)
	require.Nil(t, err)

	body := &dataBlock.Body{}
	header := &dataBlock.Header{
		Round:     50,
		TimeStamp: 5040,
	}

	pool := &outport.Pool{
		Logs: []*coreData.LogData{
			{
				TxHash: "h1",
				LogHandler: &transaction.Log{
					Events: []*transaction.Event{
						{
							Address:    []byte("addr"),
							Identifier: []byte("issueSemiFungible"),
							Topics:     [][]byte{[]byte("SEMI-abcd"), []byte("SEMI-token"), []byte("SEM"), []byte(core.SemiFungibleESDT)},
						},
						nil,
					},
				},
			},
		},
	}

	err = esProc.SaveTransactions(body, header, pool, map[string]*outport.AlteredAccount{}, false)
	require.Nil(t, err)

	ids := []string{"SEMI-abcd"}
	genericResponse := &GenericResponse{}
	err = esClient.DoMultiGet(ids, indexerdata.TokensIndex, true, genericResponse)
	require.Nil(t, err)
	require.JSONEq(t, readExpectedResult("./testdata/accountsESDTWithTokenType/token-after-issue.json"), string(genericResponse.Docs[0].Source))

	// ################ CREATE SEMI FUNGIBLE TOKEN ##########################
	shardCoordinator = &mock.ShardCoordinatorMock{
		SelfID: 0,
	}

	addr := "aaaabbbb"
	encodedAddr := hex.EncodeToString([]byte(addr))
	coreAlteredAccounts := map[string]*outport.AlteredAccount{
		encodedAddr: {
			Address: encodedAddr,
			Balance: "1000",
			Tokens: []*outport.AccountTokenData{
				{
					Identifier: "SEMI-abcd",
					Balance:    "1000",
					Nonce:      2,
					Properties: "ok",
					MetaData: &esdt.MetaData{
						Creator: []byte("creator"),
					},
				},
			},
		},
	}
	esProc, err = CreateElasticProcessor(esClient, shardCoordinator)
	require.Nil(t, err)

	header = &dataBlock.Header{
		Round:     51,
		TimeStamp: 5600,
	}

	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Creator: []byte("creator"),
		},
	}
	esdtDataBytes, _ := json.Marshal(esdtData)

	pool = &outport.Pool{
		Logs: []*coreData.LogData{
			{
				TxHash: "h1",
				LogHandler: &transaction.Log{
					Events: []*transaction.Event{
						{
							Address:    []byte("aaaabbbb"),
							Identifier: []byte(core.BuiltInFunctionESDTNFTCreate),
							Topics:     [][]byte{[]byte("SEMI-abcd"), big.NewInt(2).Bytes(), big.NewInt(1).Bytes(), esdtDataBytes},
						},
						nil,
					},
				},
			},
		},
	}

	err = esProc.SaveTransactions(body, header, pool, coreAlteredAccounts, false)
	require.Nil(t, err)

	ids = []string{"6161616162626262-SEMI-abcd-02"}
	genericResponse = &GenericResponse{}
	err = esClient.DoMultiGet(ids, indexerdata.AccountsESDTIndex, true, genericResponse)
	require.Nil(t, err)
	require.JSONEq(t, readExpectedResult("./testdata/accountsESDTWithTokenType/account-esdt.json"), string(genericResponse.Docs[0].Source))

}

func TestIndexAccountESDTWithTokenTypeShardFirstAndMetachainAfter(t *testing.T) {
	setLogLevelDebug()

	esClient, err := createESClient(esURL)
	require.Nil(t, err)

	// ################ CREATE SEMI FUNGIBLE TOKEN ##########################
	shardCoordinator := &mock.ShardCoordinatorMock{
		SelfID: 0,
	}

	body := &dataBlock.Body{}

	addr := "aaaabbbb"
	encodedAddr := hex.EncodeToString([]byte(addr))
	coreAlteredAccounts := map[string]*outport.AlteredAccount{
		encodedAddr: {
			Address: encodedAddr,
			Balance: "1000",
			Tokens: []*outport.AccountTokenData{
				{
					Identifier: "TTTT-abcd",
					Nonce:      2,
					Balance:    "1000",
					Properties: "ok",
					MetaData: &esdt.MetaData{
						Creator: []byte("creator"),
					},
				},
			},
		},
	}
	esProc, err := CreateElasticProcessor(esClient, shardCoordinator)
	require.Nil(t, err)

	header := &dataBlock.Header{
		Round:     51,
		TimeStamp: 5600,
	}

	esdtData := &esdt.ESDigitalToken{
		TokenMetaData: &esdt.MetaData{
			Creator: []byte("creator"),
		},
	}
	esdtDataBytes, _ := json.Marshal(esdtData)

	pool := &outport.Pool{
		Logs: []*coreData.LogData{
			{
				TxHash: "h1",
				LogHandler: &transaction.Log{
					Events: []*transaction.Event{
						{
							Address:    []byte("aaaabbbb"),
							Identifier: []byte(core.BuiltInFunctionESDTNFTCreate),
							Topics:     [][]byte{[]byte("TTTT-abcd"), big.NewInt(2).Bytes(), big.NewInt(1).Bytes(), esdtDataBytes},
						},
						nil,
					},
				},
			},
		},
	}

	err = esProc.SaveTransactions(body, header, pool, coreAlteredAccounts, false)
	require.Nil(t, err)

	ids := []string{"6161616162626262-TTTT-abcd-02"}
	genericResponse := &GenericResponse{}
	err = esClient.DoMultiGet(ids, indexerdata.AccountsESDTIndex, true, genericResponse)
	require.Nil(t, err)
	require.JSONEq(t, readExpectedResult("./testdata/accountsESDTWithTokenType/account-esdt-without-type.json"), string(genericResponse.Docs[0].Source))

	time.Sleep(time.Second)

	// ################ ISSUE NON FUNGIBLE TOKEN ##########################
	shardCoordinator = &mock.ShardCoordinatorMock{
		SelfID: core.MetachainShardId,
	}
	header = &dataBlock.Header{
		Round:     50,
		TimeStamp: 5040,
	}

	esProc, err = CreateElasticProcessor(esClient, shardCoordinator)
	require.Nil(t, err)

	pool = &outport.Pool{
		Logs: []*coreData.LogData{
			{
				TxHash: "h1",
				LogHandler: &transaction.Log{
					Events: []*transaction.Event{
						{
							Address:    []byte("addr"),
							Identifier: []byte("issueSemiFungible"),
							Topics:     [][]byte{[]byte("TTTT-abcd"), []byte("TTTT-token"), []byte("SEM"), []byte(core.SemiFungibleESDT)},
						},
						nil,
					},
				},
			},
		},
	}

	err = esProc.SaveTransactions(body, header, pool, map[string]*outport.AlteredAccount{}, false)
	require.Nil(t, err)

	ids = []string{"TTTT-abcd"}
	genericResponse = &GenericResponse{}
	err = esClient.DoMultiGet(ids, indexerdata.TokensIndex, true, genericResponse)
	require.Nil(t, err)
	require.JSONEq(t, readExpectedResult("./testdata/accountsESDTWithTokenType/semi-fungible-token.json"), string(genericResponse.Docs[0].Source))

	ids = []string{"6161616162626262-TTTT-abcd-02"}
	genericResponse = &GenericResponse{}
	err = esClient.DoMultiGet(ids, indexerdata.AccountsESDTIndex, true, genericResponse)
	require.Nil(t, err)
	require.JSONEq(t, readExpectedResult("./testdata/accountsESDTWithTokenType/account-esdt-with-type.json"), string(genericResponse.Docs[0].Source))

	ids = []string{"TTTT-abcd-02"}
	genericResponse = &GenericResponse{}
	err = esClient.DoMultiGet(ids, indexerdata.TokensIndex, true, genericResponse)
	require.Nil(t, err)
	require.JSONEq(t, readExpectedResult("./testdata/accountsESDTWithTokenType/semi-fungible-token-after-create.json"), string(genericResponse.Docs[0].Source))
}
