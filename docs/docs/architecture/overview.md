---
sidebar_position: 1
---

# Architecture Overview

Open-Orch is structured as a single stateless binary backed by PostgreSQL and Redis. It can run multiple instances — the reconciler and event processor coordinate via row-level locks and Redis leases.

## Component diagram

```mermaid
C4Context
    title Open-Orch System Context

    Person(dev, "Developer", "Pushes code, opens PRs")
    System_Ext(github, "GitHub", "Source of events via webhooks")
    System_Ext(buddy, "Buddy.works", "Builds Docker images")
    System_Ext(cloudflare, "Cloudflare DNS", "Wildcard DNS for preview URLs")

    System_Boundary(orch, "Open-Orch") {
        Container(api, "API Server", "Go/chi", "REST API + webhook receiver")
        Container(proc, "Event Processor", "Go", "Processes persisted webhook queue")
        Container(rec, "Reconciler", "Go", "Converges Docker state to desired topology")
        Container(cleaner, "TTL Cleaner", "Go", "Marks idle environments for destruction")
        ContainerDb(pg, "PostgreSQL", "DB", "Source of truth for all state")
        ContainerDb(redis, "Redis", "Cache", "Distributed locks and leases")
    }

    System_Ext(docker, "Docker Engine", "Runs preview containers")
    System_Ext(traefik, "Traefik", "Routes HTTPS traffic to containers")

    Rel(dev, github, "Push / open PR")
    Rel(github, api, "Webhook (HMAC-signed)")
    Rel(api, pg, "Persist events + state")
    Rel(proc, pg, "Poll FOR UPDATE SKIP LOCKED")
    Rel(proc, rec, "Trigger reconcile via state bump")
    Rel(rec, buddy, "Trigger pipeline / poll status")
    Rel(rec, docker, "EnsureNetwork, RunContainer")
    Rel(rec, traefik, "Dynamic labels on containers")
    Rel(traefik, cloudflare, "DNS-01 ACME challenge")
```

## Internal package structure

```mermaid
graph TD
    main["cmd/orchestrator"] --> api
    main --> events
    main --> reconciler
    main --> config

    api["internal/api"] --> db
    api --> lifecycle
    api --> topology
    api --> locks

    events["internal/events"] --> db
    events --> topology
    events --> locks

    reconciler["internal/reconciler"] --> db
    reconciler --> docker
    reconciler --> buddy
    reconciler --> traefik
    reconciler --> locks

    topology["internal/topology"] --> db
    topology --> normalizer

    db["internal/db"] --> models
    docker["internal/docker"]
    buddy["internal/buddy"]
    traefik["internal/traefik"]
    lifecycle["internal/lifecycle"] --> models
    locks["internal/locks"]
    normalizer["internal/normalizer"]
    models["internal/models"]
    config["internal/config"]
```

## Concurrency model

All long-running goroutines share a single `context.Context` cancelled on `SIGTERM`. They communicate exclusively via PostgreSQL — no in-process channels.

| Goroutine | Interval | Lock |
|-----------|----------|------|
| Event Processor | 2s poll | `FOR UPDATE SKIP LOCKED` on `events` table |
| Reconciler | 15s per env | Redis lease per `environment.id` |
| TTL Cleaner | 60s | No lock needed (idempotent UPDATE) |
| HTTP Server | — | Stateless |
