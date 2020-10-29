package interim

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verifa/bubbly/api/core"
	"github.com/zclconf/go-cty/cty"
)

func TestNewDB(t *testing.T) {
	zooTables := []core.Table{
		{
			Name: "zoo",
			Fields: []core.TableField{
				{
					Name:   "Name",
					Unique: true,
					Type:   cty.String,
				},
				{
					Name: "State",
					Type: cty.String,
				},
			},
			Tables: []core.Table{
				{
					Name: "restaraunts",
					Fields: []core.TableField{
						{
							Name:   "Name",
							Unique: true,
							Type:   cty.String,
						},
						{
							Name:   "Capacity",
							Unique: false,
							Type:   cty.Number,
						},
					},
				},
				{
					Name: "shops",
					Fields: []core.TableField{
						{
							Name:   "Name",
							Unique: true,
							Type:   cty.String,
						},
						{
							Name:   "Open",
							Unique: false,
							Type:   cty.Bool,
						},
					},
				},
				{
					Name: "mammals",
					Tables: []core.Table{
						{
							Name: "giraffes",
							Fields: []core.TableField{
								{
									Name:   "Name",
									Unique: true,
									Type:   cty.String,
								},
								{
									Name:   "NumSpots",
									Unique: false,
									Type:   cty.Number,
								},
							},
						},
						{
							Name: "elephants",
							Fields: []core.TableField{
								{
									Name:   "Name",
									Unique: true,
									Type:   cty.String,
								},
								{
									Name:   "Happy",
									Unique: false,
									Type:   cty.Bool,
								},
							},
						},
					},
				},
				{
					Name: "reptiles",
					Tables: []core.Table{
						{
							Name: "crocodiles",
							Fields: []core.TableField{
								{
									Name:   "Name",
									Unique: true,
									Type:   cty.String,
								},
								{
									Name:   "NumTeeth",
									Unique: false,
									Type:   cty.Number,
								},
							},
						},
					},
				},
			},
		},
	}

	db, err := NewDB(zooTables)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	zooData := core.DataBlocks{
		{
			TableName: "zoo",
			Fields: core.DataFields{
				{
					Name:  "Name",
					Value: cty.StringVal("Boise Zoo"),
				},
				{
					Name:  "State",
					Value: cty.StringVal("ID"),
				},
			},
		},
	}

	assert.NoError(t, db.Import(zooData))

	cases := []struct {
		desc    string
		query   string
		outJSON string
	}{
		{
			desc: "basic",
			query: `{
				zoo(State:"ID") {
					Name
					State
				}		
			}`,
			outJSON: `{"zoo":{"Name":"Boise Zoo","State":"ID"}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			n, err := db.Query(c.query)
			assert.NoError(t, err)
			nJSON, err := json.Marshal(n)
			assert.NoError(t, err)
			assert.Equal(t, c.outJSON, string(nJSON))
		})
	}
}
