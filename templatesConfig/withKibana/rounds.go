package withKibana

var Rounds = Object{
	"index_patterns": Array{
		"rounds-*",
	},
	"settings": Object{
		"number_of_shards":                                 3,
		"number_of_replicas":                               0,
		"opendistro.index_state_management.policy_id":      "rounds_policy",
		"opendistro.index_state_management.rollover_alias": "rounds",
	},
}
