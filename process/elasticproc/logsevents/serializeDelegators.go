package logsevents

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ElrondNetwork/elastic-indexer-go/data"
	"github.com/ElrondNetwork/elastic-indexer-go/process/elasticproc/converters"
)

// SerializeDelegators will serialize the provided delegators in a way that Elasticsearch expects a bulk request
func (lep *logsAndEventsProcessor) SerializeDelegators(delegators map[string]*data.Delegator, buffSlice *data.BufferSlice, index string) error {
	for _, delegator := range delegators {
		meta, serializedData, err := lep.prepareSerializedDelegator(delegator, index)
		if err != nil {
			return err
		}

		err = buffSlice.PutData(meta, serializedData)
		if err != nil {
			return err
		}
	}

	return nil
}

func (lep *logsAndEventsProcessor) prepareSerializedDelegator(delegator *data.Delegator, index string) ([]byte, []byte, error) {
	id := lep.computeDelegatorID(delegator)
	if delegator.ShouldDelete {
		meta := []byte(fmt.Sprintf(`{ "delete" : { "_index": "%s", "_id" : "%s" } }%s`, index, converters.JsonEscape(id), "\n"))
		return meta, nil, nil
	}

	delegatorSerialized, errMarshal := json.Marshal(delegator)
	if errMarshal != nil {
		return nil, nil, errMarshal
	}

	meta := []byte(fmt.Sprintf(`{ "update" : { "_index":"%s", "_id" : "%s" } }%s`, index, converters.JsonEscape(id), "\n"))
	if delegator.UnDelegateInfo != nil {
		serializedData, err := prepareSerializedDataForUnDelegate(delegator, delegatorSerialized)
		return meta, serializedData, err
	}
	if delegator.WithdrawFundIDs != nil {
		serializedData := prepareSerializedDataForWithdrawal(delegator, delegatorSerialized)
		return meta, serializedData, nil
	}

	return meta, prepareSerializedDataForDelegator(delegatorSerialized), nil
}

func prepareSerializedDataForDelegator(delegatorSerialized []byte) []byte {
	codeToExecute := `
		if ('create' == ctx.op) {
			ctx._source = params.delegator
		} else {
			ctx._source.activeStake = params.delegator.activeStake;
			ctx._source.activeStakeNum = params.delegator.activeStakeNum;
		}
`
	serializedDataStr := fmt.Sprintf(`{"scripted_upsert": true, "script": {`+
		`"source": "%s",`+
		`"lang": "painless",`+
		`"params": { "delegator": %s }},`+
		`"upsert": {}}`,
		converters.FormatPainlessSource(codeToExecute), string(delegatorSerialized),
	)

	return []byte(serializedDataStr)
}

func prepareSerializedDataForUnDelegate(delegator *data.Delegator, delegatorSerialized []byte) ([]byte, error) {
	unDelegateInfoSerialized, err := json.Marshal(delegator.UnDelegateInfo)
	if err != nil {
		return nil, err
	}

	codeToExecute := `
		if ('create' == ctx.op) {
			ctx._source = params.delegator
		} else {
			if (!ctx._source.containsKey('unDelegateInfo'))  
				ctx._source.unDelegateInfo = [params.unDelegate]
			} else {
				ctx._source.unDelegateInfo.add(params.unDelegate)
			}

			ctx._source.activeStake = params.delegator.activeStake;
			ctx._source.activeStakeNum = params.delegator.activeStakeNum;
		}
`
	serializedDataStr := fmt.Sprintf(`{"scripted_upsert": true, "script": {`+
		`"source": "%s",`+
		`"lang": "painless",`+
		`"params": { "delegator": %s, "unDelegate": %s }},`+
		`"upsert": {}}`,
		converters.FormatPainlessSource(codeToExecute), string(delegatorSerialized), string(unDelegateInfoSerialized),
	)

	return []byte(serializedDataStr), nil
}

func prepareSerializedDataForWithdrawal(delegator *data.Delegator, delegatorSerialized []byte) []byte {
	codeToExecute := `
		if ('create' == ctx.op) {
			ctx._source = params.delegator
		} else {
			if (!ctx._source.containsKey('unDelegateInfo'))  
				return
			} else {
				for (int i = 0; i < ctx._source.unDelegateInfo.length; i++) {
				  for (int j = 0; j < params.withdrawIds.length; j++) {
				    if 	(ctx._source.unDelegateInfo[i].id == params.withdrawIds[j]) {
						ctx._source.unDelegateInfo.remove(i);
					}
				  }
				}
			}

			ctx._source.activeStake = params.delegator.activeStake;
			ctx._source.activeStakeNum = params.delegator.activeStakeNum;
		}
`
	serializedDataStr := fmt.Sprintf(`{"scripted_upsert": true, "script": {`+
		`"source": "%s",`+
		`"lang": "painless",`+
		`"params": { "delegator": %s, "withdrawIds": %s }},`+
		`"upsert": {}}`,
		converters.FormatPainlessSource(codeToExecute), string(delegatorSerialized), delegator.WithdrawFundIDs,
	)

	return []byte(serializedDataStr)
}

func (lep *logsAndEventsProcessor) computeDelegatorID(delegator *data.Delegator) string {
	delegatorContract := delegator.Address + delegator.Contract

	hashBytes := lep.hasher.Compute(delegatorContract)

	return base64.StdEncoding.EncodeToString(hashBytes)
}
