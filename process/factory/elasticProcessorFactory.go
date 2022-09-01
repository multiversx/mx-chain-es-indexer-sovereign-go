package factory

import (
	"github.com/ElrondNetwork/elastic-indexer-go/process/dataindexer"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/accounts"
	blockProc "github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/block"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/converters"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/logsevents"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/miniblocks"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/operations"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/statistics"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/templatesAndPolicies"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/transactions"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/validators"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
)

// ArgElasticProcessorFactory is struct that is used to store all components that are needed to create an elastic processor factory
type ArgElasticProcessorFactory struct {
	Marshalizer              marshal.Marshalizer
	Hasher                   hashing.Hasher
	AddressPubkeyConverter   core.PubkeyConverter
	ValidatorPubkeyConverter core.PubkeyConverter
	DBClient                 elasticproc.DatabaseClientHandler
	ShardCoordinator         dataindexer.ShardCoordinator
	EnabledIndexes           []string
	Denomination             int
	BulkRequestMaxSize       int
	IsInImportDBMode         bool
	UseKibana                bool
}

// CreateElasticProcessor will create a new instance of ElasticProcessor
func CreateElasticProcessor(arguments ArgElasticProcessorFactory) (dataindexer.ElasticProcessor, error) {
	templatesAndPoliciesReader := templatesAndPolicies.CreateTemplatesAndPoliciesReader(arguments.UseKibana)
	indexTemplates, indexPolicies, err := templatesAndPoliciesReader.GetElasticTemplatesAndPolicies()
	if err != nil {
		return nil, err
	}

	enabledIndexesMap := make(map[string]struct{})
	for _, index := range arguments.EnabledIndexes {
		enabledIndexesMap[index] = struct{}{}
	}
	if len(enabledIndexesMap) == 0 {
		return nil, dataindexer.ErrEmptyEnabledIndexes
	}

	balanceConverter, err := converters.NewBalanceConverter(arguments.Denomination)
	if err != nil {
		return nil, err
	}

	accountsProc, err := accounts.NewAccountsProcessor(
		arguments.AddressPubkeyConverter,
		balanceConverter,
		arguments.ShardCoordinator.SelfId(),
	)
	if err != nil {
		return nil, err
	}

	blockProcHandler, err := blockProc.NewBlockProcessor(arguments.Hasher, arguments.Marshalizer)
	if err != nil {
		return nil, err
	}

	miniblocksProc, err := miniblocks.NewMiniblocksProcessor(arguments.ShardCoordinator.SelfId(), arguments.Hasher, arguments.Marshalizer, arguments.IsInImportDBMode)
	if err != nil {
		return nil, err
	}
	validatorsProc, err := validators.NewValidatorsProcessor(arguments.ValidatorPubkeyConverter, arguments.BulkRequestMaxSize)
	if err != nil {
		return nil, err
	}

	generalInfoProc := statistics.NewStatisticsProcessor()

	argsTxsProc := &transactions.ArgsTransactionProcessor{
		AddressPubkeyConverter: arguments.AddressPubkeyConverter,
		ShardCoordinator:       arguments.ShardCoordinator,
		Hasher:                 arguments.Hasher,
		Marshalizer:            arguments.Marshalizer,
		IsInImportMode:         arguments.IsInImportDBMode,
	}
	txsProc, err := transactions.NewTransactionsProcessor(argsTxsProc)
	if err != nil {
		return nil, err
	}

	argsLogsAndEventsProc := &logsevents.ArgsLogsAndEventsProcessor{
		ShardCoordinator: arguments.ShardCoordinator,
		PubKeyConverter:  arguments.AddressPubkeyConverter,
		Marshalizer:      arguments.Marshalizer,
		BalanceConverter: balanceConverter,
		Hasher:           arguments.Hasher,
	}
	logsAndEventsProc, err := logsevents.NewLogsAndEventsProcessor(argsLogsAndEventsProc)
	if err != nil {
		return nil, err
	}

	operationsProc, err := operations.NewOperationsProcessor(arguments.IsInImportDBMode, arguments.ShardCoordinator)
	if err != nil {
		return nil, err
	}

	args := &elasticproc.ArgElasticProcessor{
		BulkRequestMaxSize: arguments.BulkRequestMaxSize,
		TransactionsProc:   txsProc,
		AccountsProc:       accountsProc,
		BlockProc:          blockProcHandler,
		MiniblocksProc:     miniblocksProc,
		ValidatorsProc:     validatorsProc,
		StatisticsProc:     generalInfoProc,
		LogsAndEventsProc:  logsAndEventsProc,
		DBClient:           arguments.DBClient,
		EnabledIndexes:     enabledIndexesMap,
		UseKibana:          arguments.UseKibana,
		IndexTemplates:     indexTemplates,
		IndexPolicies:      indexPolicies,
		SelfShardID:        arguments.ShardCoordinator.SelfId(),
		OperationsProc:     operationsProc,
	}

	return elasticproc.NewElasticProcessor(args)
}
