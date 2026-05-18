// Package deps builds a directed dependency graph between repositories
// (services). It exposes a topo sort so the deployment planner can deploy
// dependencies before dependents.
package deps

import (
	"context"
	"fmt"

	"github.com/open-orch/backend/internal/db"
	"github.com/open-orch/backend/internal/models"
	"github.com/google/uuid"
)

type Graph struct {
	// Nodes by ID.
	Nodes map[uuid.UUID]models.Repository
	// Edges: from -> list of dependencies (from depends on these).
	Edges map[uuid.UUID][]uuid.UUID
}

// Build loads repos and dependencies from the DB into an in-memory graph.
func Build(ctx context.Context, repoStore *db.RepositoryStore) (*Graph, error) {
	repos, err := repoStore.List(ctx)
	if err != nil {
		return nil, err
	}
	edges, err := repoStore.Dependencies(ctx)
	if err != nil {
		return nil, err
	}
	g := &Graph{
		Nodes: map[uuid.UUID]models.Repository{},
		Edges: edges,
	}
	for _, r := range repos {
		g.Nodes[r.ID] = r
	}
	return g, nil
}

// TopoSort returns repository IDs in dependency-first order. Repositories
// with no dependencies come first. Returns an error if a cycle is detected.
func (g *Graph) TopoSort() ([]uuid.UUID, error) {
	indeg := map[uuid.UUID]int{}
	for id := range g.Nodes {
		indeg[id] = 0
	}
	// For each "a depends on b", b has an incoming edge from a's perspective?
	// We want deps first. So increment indegree of the dependent (a) for each b.
	for from, tos := range g.Edges {
		for range tos {
			indeg[from]++
		}
	}
	var ready []uuid.UUID
	for id, d := range indeg {
		if d == 0 {
			ready = append(ready, id)
		}
	}
	var out []uuid.UUID
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		out = append(out, n)
		// For each node m that depends on n, decrement m's indegree.
		for from, tos := range g.Edges {
			for _, t := range tos {
				if t == n {
					indeg[from]--
					if indeg[from] == 0 {
						ready = append(ready, from)
					}
				}
			}
		}
	}
	if len(out) != len(g.Nodes) {
		return nil, fmt.Errorf("dependency cycle detected")
	}
	return out, nil
}

// Participants returns the set of repositories that should participate in a
// preview environment given a "root" repository (e.g. the one whose PR
// triggered the env). For a multi-repo preview we currently bring up the
// whole graph; this method is the seam to constrain to transitive deps later.
func (g *Graph) Participants(root uuid.UUID) []uuid.UUID {
	// For now: all registered repos participate.
	out := make([]uuid.UUID, 0, len(g.Nodes))
	for id := range g.Nodes {
		out = append(out, id)
	}
	return out
}
