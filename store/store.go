package store

import (
	"fmt"
	"sync"

	"github.com/graphql-go/graphql"
	"github.com/verifa/bubbly/api/core"
	"github.com/zclconf/go-cty/cty"
)

// New creates a new Store for the given config.
func New(cfg Config) (*Store, error) {
	var (
		p   provider
		err error
	)

	switch cfg.Provider {
	case Postgres:
		p, err = newPostgres(cfg)
	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &Store{
		p: p,
	}, nil
}

// Store provides access to persisted readiness data.
type Store struct {
	p provider

	mu     sync.RWMutex
	schema graphql.Schema
}

// Schema gets the graphql schema for the store.
func (s *Store) Schema() graphql.Schema {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.schema
}

// Query queries the store.
func (s *Store) Query(query string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := graphql.Do(graphql.Params{
		Schema:        s.schema,
		RequestString: query,
	})

	if res.HasErrors() {
		return nil, fmt.Errorf("failed to execute query: %v", res.Errors)
	}

	return res.Data, nil
}

// Create creates a schema corresponding to a set of tables.
func (s *Store) Create(tables []core.Table) error {
	tables = addImplicitIDs(nil, tables)
	if err := s.p.Create(tables); err != nil {
		return fmt.Errorf("failed to create in provider: %w", err)
	}

	schema, err := newGraphQLSchema(tables, s.p)
	if err != nil {
		return fmt.Errorf("falied to build GraphQL schema: %w", err)
	}

	s.mu.Lock()
	s.schema = schema
	s.mu.Unlock()

	return nil
}

// Save saves data into the store.
func (s *Store) Save(data core.DataBlocks) error {
	tables, err := s.p.Save(data)
	if err != nil {
		return fmt.Errorf("falied to save data in provider: %w", err)
	}

	schema, err := newGraphQLSchema(tables, s.p)
	if err != nil {
		return fmt.Errorf("falied to build GraphQL schema: %w", err)
	}

	s.mu.Lock()
	s.schema = schema
	s.mu.Unlock()

	return nil
}

func addImplicitIDs(parent *core.Table, tables []core.Table) []core.Table {
	// We are adding at least one field (id) and possibly
	// another (parent=_id) so pad this out.
	altTables := make([]core.Table, 0, len(tables)+2)
	for _, t := range tables {
		t.Fields = append(t.Fields, core.TableField{
			Name: idFieldName,
			Type: cty.Number,
		})
		if parent != nil {
			var (
				parentIDName = parent.Name + "_id"
				hasParentID  bool
			)
			for _, f := range t.Fields {
				if f.Name == parentIDName {
					hasParentID = true
				}
			}
			if !hasParentID {
				t.Fields = append(t.Fields, core.TableField{
					Name: parentIDName,
					Type: cty.Number,
				})
			}
		}

		t.Tables = addImplicitIDs(&t, t.Tables)
		altTables = append(altTables, t)
	}
	return altTables
}

type typeInfo struct {
	ID     int64
	Tables []core.Table
}