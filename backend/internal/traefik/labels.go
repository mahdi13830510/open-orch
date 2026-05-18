// Package traefik generates the Docker labels Traefik reads to build
// dynamic routing config. Traefik runs with the Docker provider; it watches
// the daemon and discovers routes from these labels.
package traefik

import (
	"fmt"

	"github.com/open-orch/backend/internal/models"
)

// Labels returns a label map for one running container that ties it into
// Traefik routing. The orchestrator is responsible for putting these on
// containers it creates (directly or via Buddy build outputs).
//
// We assume Traefik is configured with:
//   - entrypoints: web (80) and websecure (443)
//   - a wildcard DNS-01 certResolver named "letsencrypt"
//   - the docker provider with exposedbydefault=false
func Labels(envShortID string, svc models.ServicePlan, domain *models.DomainPlan) map[string]string {
	lbl := map[string]string{
		"open-orch/env":     envShortID,
		"open-orch/service": svc.RepositoryName,
		"open-orch/commit":  svc.CommitSHA,
		"open-orch/branch":  svc.Branch,
	}
	if domain == nil {
		return lbl
	}

	router := fmt.Sprintf("%s-%s", envShortID, svc.RepositoryName)
	service := router
	port := *svc.ExposePort

	lbl["traefik.enable"] = "true"
	lbl[fmt.Sprintf("traefik.http.routers.%s.rule", router)] = fmt.Sprintf("Host(`%s`)", domain.Hostname)
	lbl[fmt.Sprintf("traefik.http.routers.%s.entrypoints", router)] = "websecure"
	lbl[fmt.Sprintf("traefik.http.routers.%s.tls", router)] = "true"
	lbl[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", router)] = "letsencrypt"
	lbl[fmt.Sprintf("traefik.http.routers.%s.tls.domains[0].main", router)] = domain.Hostname
	lbl[fmt.Sprintf("traefik.http.routers.%s.tls.domains[0].sans", router)] = "*." + domain.Hostname
	lbl[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", service)] = fmt.Sprintf("%d", port)
	return lbl
}
