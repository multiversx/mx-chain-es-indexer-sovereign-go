package accounts

import (
	"encoding/hex"
	"encoding/json"
	errorsGo "errors"
	"math/big"
	"testing"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elastic-indexer-go/errors"
	"github.com/ElrondNetwork/elastic-indexer-go/mock"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data/esdt"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/marshal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccountsProcessor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		argsFunc func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter)
		exError  error
	}{
		{
			name: "NegativeDenomination",
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return -1, &mock.MarshalizerMock{}, &mock.PubkeyConverterMock{}, &mock.AccountsStub{}
			},
			exError: errors.ErrNegativeDenominationValue,
		},
		{
			name: "NilMarshalizer",
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return 11, nil, &mock.PubkeyConverterMock{}, &mock.AccountsStub{}
			},
			exError: errors.ErrNilMarshalizer,
		},
		{
			name: "NilPubKeyConverter",
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return 11, &mock.MarshalizerMock{}, nil, &mock.AccountsStub{}
			},
			exError: errors.ErrNilPubkeyConverter,
		},
		{
			name: "NilAccounts",
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return 11, &mock.MarshalizerMock{}, &mock.PubkeyConverterMock{}, nil
			},
			exError: errors.ErrNilAccountsDB,
		},
		{
			name: "ShouldWork",
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return 11, &mock.MarshalizerMock{}, &mock.PubkeyConverterMock{}, &mock.AccountsStub{}
			},
			exError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAccountsProcessor(tt.argsFunc())
			require.True(t, errorsGo.Is(err, tt.exError))
		})
	}
}

func TestAccountsProcessor_GetAccountsWithNil(t *testing.T) {
	t.Parallel()

	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), &mock.AccountsStub{})

	regularAccounts, esdtAccounts := ap.GetAccounts(nil)
	require.Len(t, regularAccounts, 0)
	require.Len(t, esdtAccounts, 0)
}

func TestAccountsProcessor_PrepareRegularAccountsMapWithNil(t *testing.T) {
	t.Parallel()

	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), &mock.AccountsStub{})

	accountsInfo := ap.PrepareRegularAccountsMap(nil)
	require.Len(t, accountsInfo, 0)
}

func TestAccountsProcessor_ComputeBalanceAsFloat(t *testing.T) {
	t.Parallel()

	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), &mock.AccountsStub{})
	require.NotNil(t, ap)

	tests := []struct {
		input  *big.Int
		output float64
	}{
		{
			input:  big.NewInt(200000000000000000),
			output: float64(20000000),
		},
		{
			input:  big.NewInt(57777777777),
			output: 5.7777777777,
		},
		{
			input:  big.NewInt(5777779),
			output: 0.0005777779,
		},
		{
			input:  big.NewInt(7),
			output: 0.0000000007,
		},
		{
			input:  big.NewInt(-7),
			output: 0.0,
		},
	}

	for _, tt := range tests {
		out := ap.computeBalanceAsFloat(tt.input)
		assert.Equal(t, tt.output, out)
	}
}

func TestGetESDTInfo_CannotRetriveValueShoudError(t *testing.T) {
	t.Parallel()

	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), &mock.AccountsStub{})
	require.NotNil(t, ap)

	localErr := errorsGo.New("local error")
	wrapAccount := &data.AccountESDT{
		Account: &mock.UserAccountStub{
			DataTrieTrackerCalled: func() state.DataTrieTracker {
				return &mock.DataTrieTrackerStub{
					RetrieveValueCalled: func(key []byte) ([]byte, error) {
						return nil, localErr
					},
				}
			},
		},
		TokenIdentifier: "token",
	}
	_, _, err := ap.getESDTInfo(wrapAccount)
	require.Equal(t, localErr, err)
}

func TestGetESDTInfo(t *testing.T) {
	t.Parallel()

	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), &mock.AccountsStub{})
	require.NotNil(t, ap)

	esdtToken := &esdt.ESDigitalToken{
		Value:      big.NewInt(1000),
		Properties: []byte("ok"),
	}

	tokenIdentifier := "token-001"
	wrapAccount := &data.AccountESDT{
		Account: &mock.UserAccountStub{
			DataTrieTrackerCalled: func() state.DataTrieTracker {
				return &mock.DataTrieTrackerStub{
					RetrieveValueCalled: func(key []byte) ([]byte, error) {
						return json.Marshal(esdtToken)
					},
				}
			},
		},
		TokenIdentifier: tokenIdentifier,
	}
	balance, prop, err := ap.getESDTInfo(wrapAccount)
	require.Nil(t, err)
	require.Equal(t, big.NewInt(1000), balance)
	require.Equal(t, hex.EncodeToString([]byte("ok")), prop)
}

func TestAccountsProcessor_GetAccountsEGLDAccounts(t *testing.T) {
	t.Parallel()

	addr := "aaaabbbb"
	mockAccount := &mock.UserAccountStub{}
	accountsStub := &mock.AccountsStub{
		LoadAccountCalled: func(container []byte) (state.AccountHandler, error) {
			return mockAccount, nil
		},
	}
	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), accountsStub)
	require.NotNil(t, ap)

	alteredAccounts := map[string]*data.AlteredAccount{
		addr: {
			IsESDTOperation: false,
			TokenIdentifier: "",
		},
	}
	accounts, esdtAccounts := ap.GetAccounts(alteredAccounts)
	require.Equal(t, 0, len(esdtAccounts))
	require.Equal(t, []*data.Account{
		{UserAccount: mockAccount},
	}, accounts)
}

func TestAccountsProcessor_GetAccountsESDTAccount(t *testing.T) {
	t.Parallel()

	addr := "aaaabbbb"
	mockAccount := &mock.UserAccountStub{}
	accountsStub := &mock.AccountsStub{
		LoadAccountCalled: func(container []byte) (state.AccountHandler, error) {
			return mockAccount, nil
		},
	}
	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), accountsStub)
	require.NotNil(t, ap)

	alteredAccounts := map[string]*data.AlteredAccount{
		addr: {
			IsESDTOperation: true,
			TokenIdentifier: "token",
		},
	}
	accounts, esdtAccounts := ap.GetAccounts(alteredAccounts)
	require.Equal(t, 0, len(accounts))
	require.Equal(t, []*data.AccountESDT{
		{Account: mockAccount, TokenIdentifier: "token"},
	}, esdtAccounts)
}

func TestAccountsProcessor_PrepareAccountsMapEGLD(t *testing.T) {
	t.Parallel()

	addr := "aaaabbbb"
	mockAccount := &mock.UserAccountStub{
		GetNonceCalled: func() uint64 {
			return 1
		},
		GetBalanceCalled: func() *big.Int {
			return big.NewInt(1000)
		},
		AddressBytesCalled: func() []byte {
			return []byte(addr)
		},
	}

	egldAccount := &data.Account{
		UserAccount: mockAccount,
		IsSender:    false,
	}

	accountsStub := &mock.AccountsStub{
		LoadAccountCalled: func(container []byte) (state.AccountHandler, error) {
			return mockAccount, nil
		},
	}
	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), accountsStub)
	require.NotNil(t, ap)

	res := ap.PrepareRegularAccountsMap([]*data.Account{egldAccount})
	require.Equal(t, map[string]*data.AccountInfo{
		hex.EncodeToString([]byte(addr)): {
			Nonce:      1,
			Balance:    "1000",
			BalanceNum: ap.computeBalanceAsFloat(big.NewInt(1000)),
		},
	}, res)
}

func TestAccountsProcessor_PrepareAccountsMapESDT(t *testing.T) {
	t.Parallel()

	esdtToken := &esdt.ESDigitalToken{
		Value:      big.NewInt(1000),
		Properties: []byte("ok"),
	}

	addr := "aaaabbbb"
	mockAccount := &mock.UserAccountStub{
		DataTrieTrackerCalled: func() state.DataTrieTracker {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return json.Marshal(esdtToken)
				},
			}
		},
		AddressBytesCalled: func() []byte {
			return []byte(addr)
		},
	}
	accountsStub := &mock.AccountsStub{
		LoadAccountCalled: func(container []byte) (state.AccountHandler, error) {
			return mockAccount, nil
		},
	}
	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), accountsStub)
	require.NotNil(t, ap)

	res := ap.PrepareAccountsMapESDT([]*data.AccountESDT{{Account: mockAccount, TokenIdentifier: "token"}})
	require.Equal(t, map[string]*data.AccountInfo{
		hex.EncodeToString([]byte(addr)): {
			Address:         hex.EncodeToString([]byte(addr)),
			Balance:         "1000",
			BalanceNum:      ap.computeBalanceAsFloat(big.NewInt(1000)),
			TokenIdentifier: "token",
			Properties:      hex.EncodeToString([]byte("ok")),
		},
	}, res)
}

func TestAccountsProcessor_PrepareAccountsHistory(t *testing.T) {
	t.Parallel()

	accounts := map[string]*data.AccountInfo{
		"addr1": {
			Address:         "addr1",
			Balance:         "112",
			TokenIdentifier: "token-112",
			IsSender:        true,
		},
	}

	ap, _ := NewAccountsProcessor(10, &mock.MarshalizerMock{}, mock.NewPubkeyConverterMock(32), &mock.AccountsStub{})

	res := ap.PrepareAccountsHistory(100, accounts)
	accountBalanceHistory := res["addr1_100"]
	require.Equal(t, &data.AccountBalanceHistory{
		Address:         "addr1",
		Timestamp:       100,
		Balance:         "112",
		TokenIdentifier: "token-112",
		IsSender:        true,
	}, accountBalanceHistory)
}

func TestAccountsProcessor_GetUserAccountErrors(t *testing.T) {
	t.Parallel()

	localErr := errorsGo.New("local error")
	tests := []struct {
		name         string
		argsFunc     func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter)
		inputAddress string
		exError      error
	}{
		{
			name:    "InvalidAddress",
			exError: localErr,
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return 10, &mock.MarshalizerMock{}, &mock.PubkeyConverterStub{
					DecodeCalled: func(humanReadable string) ([]byte, error) {
						return nil, localErr
					}}, &mock.AccountsStub{}
			},
		},
		{
			name:    "CannotLoadAccount",
			exError: localErr,
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return 10, &mock.MarshalizerMock{}, &mock.PubkeyConverterMock{}, &mock.AccountsStub{
					LoadAccountCalled: func(container []byte) (state.AccountHandler, error) {
						return nil, localErr
					},
				}
			},
		},
		{
			name:    "CannotCastAccount",
			exError: errors.ErrCannotCastAccountHandlerToUserAccount,
			argsFunc: func() (int, marshal.Marshalizer, core.PubkeyConverter, state.AccountsAdapter) {
				return 10, &mock.MarshalizerMock{}, &mock.PubkeyConverterMock{}, &mock.AccountsStub{
					LoadAccountCalled: func(container []byte) (state.AccountHandler, error) {
						return nil, nil
					},
				}
			},
		},
	}

	for _, tt := range tests {
		ap, err := NewAccountsProcessor(tt.argsFunc())
		require.Nil(t, err)

		_, err = ap.getUserAccount(tt.inputAddress)
		require.Equal(t, tt.exError, err)
	}
}
