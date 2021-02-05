package withKibana

var Rating = Object{
	"index_patterns": Array{
		"rating-*",
	},
	"settings": Object{
		"number_of_shards":                                 1,
		"number_of_replicas":                               0,
		"opendistro.index_state_management.policy_id":      "rating_policy",
		"opendistro.index_state_management.rollover_alias": "rating",
	},
	"mappings": Object{
		"properties": Object{
			"validatorsRating": Object{
				"properties": Object{
					"rating": Object{
						"type": "float",
					},
				},
			},
		},
	},
}
