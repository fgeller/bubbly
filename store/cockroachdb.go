package store

import (
	"context"
	"fmt"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v4"
	"github.com/verifa/bubbly/env"
)

func newCockroachdb(bCtx *env.BubblyContext) (*cockroachdb, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s/%s",
		bCtx.StoreConfig.CockroachUser,
		bCtx.StoreConfig.CockroachPassword,
		bCtx.StoreConfig.CockroachAddr,
		bCtx.StoreConfig.CockroachDatabase,
	)
	conn, err := psqlNewConn(bCtx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize connection to db: %w", err)
	}

	return &cockroachdb{
		conn: conn,
	}, nil
}

type cockroachdb struct {
	conn *pgx.Conn
}

func (c *cockroachdb) Apply(schema *bubblySchema) error {

	err := crdbpgx.ExecuteTx(context.Background(), c.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return psqlApplySchema(tx, schema)
	})
	if err != nil {
		return fmt.Errorf("failed to apply tables: %w", err)
	}

	return nil
}

func (c *cockroachdb) Save(schema *bubblySchema, tree dataTree) error {

	err := crdbpgx.ExecuteTx(context.Background(), c.conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		saveNode := func(node *dataNode) error {
			return psqlSaveNode(tx, node, schema)
		}

		return tree.traverse(saveNode)
	})
	if err != nil {
		return fmt.Errorf("failed to save data in cockroachdb: %w", err)
	}
	return nil
}

func (c *cockroachdb) ResolveQuery(graph *schemaGraph, params graphql.ResolveParams) (interface{}, error) {
	return psqlResolveRootQueries(c.conn, graph, params)
}
