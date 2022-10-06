package logsevents

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elastic-indexer-go/mock"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/stretchr/testify/require"
)

func TestProcessLogsAndEventsESDT_IntraShard(t *testing.T) {
	t.Parallel()

	fungibleProc := newFungibleESDTProcessor(&mock.PubkeyConverterMock{}, &mock.ShardCoordinatorMock{})

	event := &transaction.Event{
		Address:    []byte("addr"),
		Identifier: []byte(core.BuiltInFunctionESDTTransfer),
		Topics:     [][]byte{[]byte("my-token"), big.NewInt(0).Bytes(), big.NewInt(0).SetUint64(100).Bytes(), []byte("receiver")},
	}
	altered := data.NewAlteredAccounts()

	fungibleProc.processEvent(&argsProcessEvent{
		event:    event,
		accounts: altered,
	})

	alteredAddrSender, ok := altered.Get("61646472")
	require.True(t, ok)
	require.Equal(t, &data.AlteredAccount{
		IsESDTOperation: true,
		TokenIdentifier: "my-token",
	}, alteredAddrSender[0])

	alteredAddrReceiver, ok := altered.Get("7265636569766572")
	require.True(t, ok)
	require.Equal(t, &data.AlteredAccount{
		IsESDTOperation: true,
		TokenIdentifier: "my-token",
	}, alteredAddrReceiver[0])
}

func TestProcessLogsAndEventsESDT_CrossShardOnSource(t *testing.T) {
	t.Parallel()

	receiverAddr := []byte("receiver")
	fungibleProc := newFungibleESDTProcessor(&mock.PubkeyConverterMock{}, &mock.ShardCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			if bytes.Equal(address, receiverAddr) {
				return 1
			}
			return 0
		},
	})

	event := &transaction.Event{
		Address:    []byte("addr"),
		Identifier: []byte(core.BuiltInFunctionESDTTransfer),
		Topics:     [][]byte{[]byte("my-token"), big.NewInt(0).Bytes(), big.NewInt(0).SetUint64(100).Bytes(), receiverAddr},
	}

	altered := data.NewAlteredAccounts()

	res := fungibleProc.processEvent(&argsProcessEvent{
		event:    event,
		accounts: altered,
	})
	require.Equal(t, "my-token", res.identifier)
	require.Equal(t, "100", res.value)
	require.Equal(t, true, res.processed)

	alteredAddrSender, ok := altered.Get("61646472")
	require.True(t, ok)
	require.Equal(t, &data.AlteredAccount{
		IsESDTOperation: true,
		TokenIdentifier: "my-token",
	}, alteredAddrSender[0])

	_, ok = altered.Get("7265636569766572")
	require.False(t, ok)
}

func TestProcessLogsAndEventsESDT_CrossShardOnDestination(t *testing.T) {
	t.Parallel()

	senderAddr := []byte("addr")
	receiverAddr := []byte("receiver")
	fungibleProc := newFungibleESDTProcessor(&mock.PubkeyConverterMock{}, &mock.ShardCoordinatorMock{
		ComputeIdCalled: func(address []byte) uint32 {
			if bytes.Equal(address, senderAddr) {
				return 1
			}
			return 0
		},
	})

	event := &transaction.Event{
		Address:    senderAddr,
		Identifier: []byte(core.BuiltInFunctionESDTTransfer),
		Topics:     [][]byte{[]byte("my-token"), big.NewInt(0).Bytes(), big.NewInt(0).SetUint64(100).Bytes(), receiverAddr},
	}

	altered := data.NewAlteredAccounts()

	res := fungibleProc.processEvent(&argsProcessEvent{
		event:    event,
		accounts: altered,
	})
	require.Equal(t, "my-token", res.identifier)
	require.Equal(t, "100", res.value)
	require.Equal(t, true, res.processed)

	alteredAddrSender, ok := altered.Get("7265636569766572")
	require.True(t, ok)
	require.Equal(t, &data.AlteredAccount{
		IsESDTOperation: true,
		TokenIdentifier: "my-token",
	}, alteredAddrSender[0])

	_, ok = altered.Get("61646472")
	require.False(t, ok)
}

func TestNftsProcessor_processLogAndEventsFungibleESDT_Wipe(t *testing.T) {
	t.Parallel()

	nftsProc := newFungibleESDTProcessor(&mock.PubkeyConverterMock{}, &mock.ShardCoordinatorMock{})

	events := &transaction.Event{
		Address:    []byte("addr"),
		Identifier: []byte(core.BuiltInFunctionESDTWipe),
		Topics:     [][]byte{[]byte("esdt-0123"), big.NewInt(0).SetUint64(0).Bytes(), big.NewInt(0).Bytes(), []byte("receiver")},
	}

	altered := data.NewAlteredAccounts()

	res := nftsProc.processEvent(&argsProcessEvent{
		event:        events,
		accounts:     altered,
		timestamp:    10000,
		tokensSupply: data.NewTokensInfo(),
	})
	require.Equal(t, "esdt-0123", res.identifier)
	require.Equal(t, "0", res.value)
	require.Equal(t, true, res.processed)

	alteredAddrSender, ok := altered.Get("61646472")
	require.True(t, ok)
	require.Equal(t, &data.AlteredAccount{
		TokenIdentifier: "esdt-0123",
		IsESDTOperation: true,
	}, alteredAddrSender[0])

	alteredAddrReceiver, ok := altered.Get("7265636569766572")
	require.True(t, ok)
	require.Equal(t, &data.AlteredAccount{
		TokenIdentifier: "esdt-0123",
		IsESDTOperation: true,
	}, alteredAddrReceiver[0])
}
