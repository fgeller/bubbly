package store

import (
	"fmt"

	"github.com/valocode/bubbly/api/core"
	"github.com/valocode/bubbly/bubbly/builtin"
)

//
// The Schema Graph is a graph representation of the Bubbly Schema.
//

// RelType describes the relationship type of a directed edge from a --> b
type RelType int

// The difference between `OneToOne` and `BelongsTo` is in the order.
// table "A" {
//   table "B" { single = true }
//   table "C" {}
// }
// Table B belongs to A. And Table A has a OneToOne to B.
// Table C belongs to A. And Table A has a OneToMany to C.
// So the relationships describe the direction of the edge.
const (
	OneToOne RelType = iota
	OneToMany
	BelongsTo
)

// SchemaNode represents a node in the schema graph.
// A node is a wrapper around core.Table with the edges for explicit
// relationships to other nodes (and therefore tables)
type SchemaNode struct {
	Table *core.Table
	Edges SchemaEdges
}

func (n SchemaNode) Edge(node string) (*SchemaEdge, error) {
	for _, edge := range n.Edges {
		if edge.Node.Table.Name == node {
			return edge, nil
		}
	}
	return nil, fmt.Errorf("edge does not exist between nodes %s --> %s", n.Table.Name, node)
}

// nodeRefMap maps node names to the corresponding structures of type node
type nodeRefMap map[string]*SchemaNode

// schemaNodes is a list of graph nodes
type schemaNodes []*SchemaNode

// SchemaEdge represents an edge in the graph
type SchemaEdge struct {
	Node *SchemaNode
	Rel  RelType
}

// isScalar returns true if the return type from the node which this edge points
// to should be scalar. This is true, unless the edge relationship is OneToMany
func (e *SchemaEdge) isScalar() bool {
	return e.Rel != OneToMany
}

// SchemaEdges is a list graph edges
type SchemaEdges []*SchemaEdge

// SchemaGraph represents a graph created from the bubbly schema.
type SchemaGraph struct {
	Nodes schemaNodes
	// NodeIndex stores an index to the nodes using the schema table name.
	// This is probably not the best for performance, but our schemas probably
	// will not become huge and this is a great convienice for consumers of the
	// graph, to not have to traverse the graph to find a node
	NodeIndex nodeRefMap
}

// traverse applies the callback function to every node of the SchemaGraph.
func (g *SchemaGraph) Traverse(fnVisit func(node *SchemaNode) error) error {
	var visited = make(map[string]struct{})
	for _, n := range g.Nodes {
		// If we have already visited the node, then continue the loop
		if _, ok := visited[n.Table.Name]; ok {
			continue
		}
		if err := visitSchemaNode(n, visited, fnVisit); err != nil {
			return fmt.Errorf("failed to traverse schema graph: %w", err)
		}
	}
	return nil
}

// visitSchemaNode is used by traverse function to make sure a node is "visited" only once,
// that is to make sure that the callback function is applied to the node only once.
func visitSchemaNode(node *SchemaNode, visited map[string]struct{}, fnVisit func(node *SchemaNode) error) error {
	if err := fnVisit(node); err != nil {
		return err
	}
	visited[node.Table.Name] = struct{}{}

	for _, e := range node.Edges {
		// If we have already visited the node, then continue the loop
		if _, ok := visited[e.Node.Table.Name]; ok {
			continue
		}
		if err := visitSchemaNode(e.Node, visited, fnVisit); err != nil {
			return err
		}
	}
	return nil
}

// addEdgeFromJoin takes a node and creates bi-directional edges between the
// nodes. Noteworthy is the relationship that the edges describe
func (n *SchemaNode) addEdgeFromJoin(child *SchemaNode, unique bool) {
	var (
		// This node has a OneToMany or OneToOne relationship with the child node
		edgeToChild = &SchemaEdge{Node: child, Rel: OneToMany}
		// The child "BelongsTo" the parent (this nodes)
		edgeToParent = &SchemaEdge{Node: n, Rel: BelongsTo}
	)
	if unique {
		// If unique, then it's a OneToOne relationship, not OneToMany
		edgeToChild.Rel = OneToOne
	}
	// Add the edge to the child to this node
	n.Edges = append(n.Edges, edgeToChild)
	// Also add the reverse relationship
	child.Edges = append(child.Edges, edgeToParent)
}

// internalSchemaGraph returns a schema graph based on the internal tables
func internalSchemaGraph() *SchemaGraph {
	flatTables := FlattenTables(builtin.BuiltinTables, nil)
	graph, err := NewSchemaGraph(flatTables)
	if err != nil {
		// This is controlled entirely by development so no input can affect this
		// so panic as a developer has done something wrong
		panic("failed to create schema graph from internal tables")
	}
	return graph
}

// newSchemaGraphFromMap returns a new Schema Graph,
// created from the tables contained in the Bubbly Schema.
//
// It is implemented as a wrapper around newSchemaGraph for backwards
// compatibility with the current way the schema is stored in the provider.
//
// FIXME: This project is too young to have "backwards compatibility" layer!
func newSchemaGraphFromMap(tables map[string]core.Table) (*SchemaGraph, error) {
	var ts = make(core.Tables, 0, len(tables))
	for _, t := range tables {
		ts = append(ts, t)
	}
	return NewSchemaGraph(ts)
}

// NewSchemaGraph returns a new Schema Graph,
// created from the tables coming from the Bubbly Schema.
func NewSchemaGraph(tables core.Tables) (*SchemaGraph, error) {

	var (
		nodes = make(nodeRefMap)
		graph = &SchemaGraph{NodeIndex: nodes}
	)

	// Pull all the nodes from the definitions of tables.
	nodes.createFrom(tables)

	// First iterate over the top-level tables to extract the root nodes in the
	// graph. Tables at the top-level, without any joins, do not have any edges
	// going to them, so they are root nodes.
	for _, table := range tables {
		if len(table.Joins) == 0 {
			graph.Nodes = append(graph.Nodes, nodes[table.Name])
		}
	}

	// Connect related nodes based on the information from the list of tables,
	// each table in the list knows what other tables it is connected to.
	if err := nodes.connectFrom(tables, nil); err != nil {
		return graph, fmt.Errorf("failed to create graph: %w", err)
	}
	return graph, nil
}

// createFrom creates a node for every table in the given list.
func (nodes *nodeRefMap) createFrom(tables core.Tables) {
	for index, t := range tables {
		(*nodes)[t.Name] = &SchemaNode{Table: &tables[index]}
		nodes.createFrom(t.Tables)
	}
}

// connectFrom connects related nodes based on join information stored in the given list of tables
func (nodes *nodeRefMap) connectFrom(tables core.Tables, parent *SchemaNode) error {

	for _, table := range tables {
		var node = (*nodes)[table.Name]
		// Handle the explicit joins
		for _, join := range table.Joins {
			// A join indicates that this table "belongs to" another talbe,
			// i.e. this table is a child of that table
			parent, ok := (*nodes)[join.Table]
			if !ok {
				return fmt.Errorf("join refers to unknown table: %s --> %s", table.Name, join.Table)
			}
			// Create the edge from parent to node
			parent.addEdgeFromJoin(node, join.Single)
		}
		// Handle the implicit joins, i.e. a table nested within a table
		if parent != nil {
			parent.addEdgeFromJoin(node, table.Single)
		}
		// Recurse
		nodes.connectFrom(table.Tables, node)

		// Clear unnecessary data
		table.Tables = nil
		// TODO: cannot remove joins because it breaks schema diff tests
		// table.Joins = nil
	}
	return nil
}
