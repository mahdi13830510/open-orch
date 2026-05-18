// Package topology resolves a desired deployment topology for one feature.
//
// Given a feature slug like "billing-redesign", we look at every participating
// repository and decide:
//
//   - is there a matching branch in that repo? Use it.
//   - if not, apply the configured fallback strategy.
//
// The output is a Topology snapshot stored on the environment row.
package topology

import (
	"context"
	"fmt"

	"github.com/open-orch/backend/internal/config"
	"github.com/open-orch/backend/internal/db"
	"github.com/open-orch/backend/internal/deps"
	"github.com/open-orch/backend/internal/models"
	"github.com/open-orch/backend/internal/normalizer"
	"github.com/google/uuid"
)

type Resolver struct {
	Cfg      *config.Config
	Repos    *db.RepositoryStore
	PRs      *db.PullRequestStore
	Features *db.FeatureStore
}

// Resolve produces a Topology describing what should run for this feature.
// envID/shortID is required so service plans can derive container names.
func (r *Resolver) Resolve(ctx context.Context, env *models.Environment, feature *models.Feature, originPR *models.PullRequest) (*models.Topology, error) {
	graph, err := deps.Build(ctx, r.Repos)
	if err != nil {
		return nil, fmt.Errorf("build dep graph: %w", err)
	}

	// Resolve order: dependencies before dependents.
	order, err := graph.TopoSort()
	if err != nil {
		return nil, err
	}

	topo := &models.Topology{
		EnvironmentID: env.ID,
		FeatureSlug:   feature.Slug,
		Generation:    env.Generation + 1,
		GlobalEnv:     map[string]string{},
		Services:      make([]models.ServicePlan, 0, len(order)),
	}

	// First pass: pick a branch+sha for every service. We need to do this
	// before computing env vars so we can inject internal DNS-style URLs that
	// reference sibling container names.
	picks := map[uuid.UUID]pick{}
	for _, repoID := range order {
		repo := graph.Nodes[repoID]
		p, err := r.pickBranch(ctx, &repo, feature.Slug, originPR)
		if err != nil {
			return nil, err
		}
		picks[repoID] = p
	}

	// Second pass: build ServicePlans. The container name pattern is
	// "<service>-<env-short-id>", which doubles as the in-network DNS name.
	for _, repoID := range order {
		repo := graph.Nodes[repoID]
		p := picks[repoID]

		container := fmt.Sprintf("%s-%s", normalizer.HostnameLabel(repo.Name), env.ShortID)

		serviceEnv := buildServiceEnv(repo, env.ShortID, picks, graph)

		topo.Services = append(topo.Services, models.ServicePlan{
			RepositoryID:    repo.ID,
			RepositoryName:  repo.Name,
			Branch:          p.branch,
			CommitSHA:       p.sha,
			ContainerName:   container,
			ExposePort:      repo.ExposePort,
			EnvVars:         serviceEnv,
			Strategy:        p.strategy,
			HealthcheckPath: "/health",
		})
	}

	// Domains: only for services that expose a port.
	topo.Domains = r.buildDomains(env, feature, originPR, topo.Services)

	return topo, nil
}

// pick is the resolver's per-service decision.
type pick struct {
	branch   string
	sha      string
	strategy models.SelectionStrategy
}

// pickBranch implements the matching + fallback policy for one repository.
func (r *Resolver) pickBranch(ctx context.Context, repo *models.Repository, featureSlug string, originPR *models.PullRequest) (pick, error) {
	// 1. Exact match: an open PR in this repo whose normalized branch == feature slug.
	//    We look across all open PRs, normalize, compare.
	if pr, err := r.findOpenPRByFeature(ctx, repo.ID, featureSlug); err == nil && pr != nil {
		return pick{branch: pr.Branch, sha: pr.HeadSHA, strategy: models.StrategyMatched}, nil
	}

	// 2. If the origin PR belongs to *this* repo (the one that triggered things),
	//    use its branch directly even if no PR row exists.
	if originPR != nil && originPR.RepositoryID == repo.ID {
		return pick{branch: originPR.Branch, sha: originPR.HeadSHA, strategy: models.StrategyMatched}, nil
	}

	// 3. Fallback per configured strategy.
	switch r.Cfg.DefaultFallback {
	case "latest_stable":
		return pick{branch: repo.DefaultBranch, sha: "HEAD", strategy: models.StrategyFallbackLatest}, nil
	case "main":
		return pick{branch: repo.DefaultBranch, sha: "HEAD", strategy: models.StrategyFallbackMain}, nil
	case "auto_create":
		// In a fully implemented system this would call the GitHub API to
		// create a draft branch+PR. We tag the strategy and use main as base.
		return pick{branch: repo.DefaultBranch, sha: "HEAD", strategy: models.StrategyFallbackAutoCreate}, nil
	case "previous_compatible":
		return pick{branch: repo.DefaultBranch, sha: "HEAD", strategy: models.StrategyFallbackPrevious}, nil
	}
	return pick{branch: repo.DefaultBranch, sha: "HEAD", strategy: models.StrategyFallbackMain}, nil
}

// findOpenPRByFeature: scan PRs for the repo whose normalized branch matches.
func (r *Resolver) findOpenPRByFeature(ctx context.Context, repoID uuid.UUID, slug string) (*models.PullRequest, error) {
	rows, err := r.PRs.DB.Pool.Query(ctx,
		`SELECT id, repository_id, number, branch, head_sha, state, COALESCE(title,''), COALESCE(author,'')
		 FROM pull_requests WHERE repository_id=$1 AND state='open'`, repoID)
	if err != nil { return nil, err }
	defer rows.Close()
	for rows.Next() {
		var pr models.PullRequest
		if err := rows.Scan(&pr.ID, &pr.RepositoryID, &pr.Number, &pr.Branch,
			&pr.HeadSHA, &pr.State, &pr.Title, &pr.Author); err != nil {
			return nil, err
		}
		if normalizer.Branch(pr.Branch) == slug {
			return &pr, nil
		}
	}
	return nil, nil
}

// buildServiceEnv synthesizes environment variables for one service, including
// internal DNS URLs that point at sibling containers (using Docker DNS).
func buildServiceEnv(repo models.Repository, envShortID string, picks map[uuid.UUID]pick, g *deps.Graph) map[string]string {
	out := map[string]string{
		"ENVIRONMENT_ID": envShortID,
		"SERVICE_NAME":   repo.Name,
	}
	// For every dependency, inject SERVICE_URL pointing at the container.
	for _, depID := range g.Edges[repo.ID] {
		dep, ok := g.Nodes[depID]
		if !ok { continue }
		host := fmt.Sprintf("%s-%s", normalizer.HostnameLabel(dep.Name), envShortID)
		port := 80
		if dep.ExposePort != nil {
			port = *dep.ExposePort
		}
		key := fmt.Sprintf("%s_URL", upperEnv(dep.Name))
		out[key] = fmt.Sprintf("http://%s:%d", host, port)
	}
	return out
}

// buildDomains generates Traefik-managed hostnames. Pattern:
//
//	<feature>-pr<num>.<base-domain>             // for the entry service (port-exposing repo named "frontend"-like)
//	api-<feature>-pr<num>.<base-domain>         // for backend/gateway/auth that expose ports
func (r *Resolver) buildDomains(env *models.Environment, feature *models.Feature, originPR *models.PullRequest, services []models.ServicePlan) []models.DomainPlan {
	prNum := 0
	if originPR != nil {
		prNum = originPR.Number
	}
	suffix := feature.Slug
	if prNum > 0 {
		suffix = normalizer.HostnameLabel(feature.Slug, fmt.Sprintf("pr%d", prNum))
	}

	var out []models.DomainPlan
	for _, s := range services {
		if s.ExposePort == nil {
			continue
		}
		var host string
		switch s.RepositoryName {
		case "frontend", "web", "app", "ui":
			host = fmt.Sprintf("%s.%s", suffix, env.BaseDomain)
		default:
			host = fmt.Sprintf("%s-%s.%s",
				normalizer.HostnameLabel(s.RepositoryName), suffix, env.BaseDomain)
		}
		out = append(out, models.DomainPlan{
			Hostname:       host,
			RepositoryName: s.RepositoryName,
			RepositoryID:   s.RepositoryID,
			TargetPort:     *s.ExposePort,
			TLS:            true,
		})
	}
	return out
}

func upperEnv(name string) string {
	// "auth-service" -> "AUTH_SERVICE"
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z':
			out = append(out, c-32)
		case c >= 'A' && c <= 'Z' || c >= '0' && c <= '9':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
