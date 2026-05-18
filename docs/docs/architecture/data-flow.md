---
sidebar_position: 2
---

# Data Flow

## GitHub webhook → running environment

```mermaid
sequenceDiagram
    participant GH as GitHub
    participant API as API Server
    participant DB as PostgreSQL
    participant EP as Event Processor
    participant Rec as Reconciler
    participant Buddy as Buddy.works
    participant Docker as Docker Engine
    participant Traefik as Traefik

    GH->>API: POST /webhooks/github (HMAC-signed)
    API->>API: Verify HMAC signature
    API->>DB: INSERT INTO events (idempotent on delivery_id)
    API-->>GH: 202 Accepted

    loop Every 2s
        EP->>DB: SELECT ... FOR UPDATE SKIP LOCKED
        DB-->>EP: Unprocessed event batch
        EP->>EP: Normalize branch → feature slug
        EP->>DB: Upsert PullRequest
        EP->>DB: GetOrCreate Feature
        EP->>EP: Resolve topology (pick branches per repo)
        EP->>DB: BumpGeneration + UPDATE state=resolving
        EP->>DB: Mark events processed
    end

    loop Every 15s per environment
        Rec->>DB: SELECT envs WHERE state != destroyed
        Rec->>Redis: Acquire lease (env_id)
        Rec->>Docker: EnsureNetwork(env.docker_network)

        loop Per ServicePlan in topology
            Rec->>DB: Get existing Deployment
            alt No existing deployment
                Rec->>Buddy: TriggerPipeline(repo, branch)
                Rec->>DB: INSERT Deployment(state=building)
            else Deployment building
                Rec->>Buddy: GetRun(run_id)
                alt Run succeeded
                    Rec->>Docker: PullImage
                    Rec->>Docker: RunContainer + Traefik labels
                    Rec->>DB: UPDATE Deployment(state=running)
                    Rec->>DB: Upsert RuntimeResource
                end
            end
        end

        Rec->>DB: SyncDomains
        Rec->>DB: UPDATE env state (healthy/degraded/failed)
        Rec->>Redis: Release lease
    end
```

## Environment destroy flow

```mermaid
sequenceDiagram
    participant Client as API Client
    participant API as API Server
    participant DB as PostgreSQL
    participant Rec as Reconciler
    participant Docker as Docker Engine

    Client->>API: DELETE /environments/{id}
    API->>API: Check FSM transition (→ destroying)
    API->>DB: UPDATE environments SET state=destroying
    API-->>Client: 202 Accepted

    loop Next reconciler tick
        Rec->>DB: Find env WHERE state=destroying
        loop Per container in RuntimeResources
            Rec->>Docker: StopContainer
            Rec->>Docker: RemoveContainer
            Rec->>DB: Delete RuntimeResource
        end
        Rec->>Docker: RemoveNetwork
        Rec->>DB: Delete Domains
        Rec->>DB: UPDATE env SET state=destroyed, destroyed_at=NOW()
    end
```

## Branch resolution strategy

When multiple repos share a feature branch, the topology resolver picks the best available image per repo:

```mermaid
flowchart TD
    start([Resolve branch for repo]) --> check_pr{Matching PR exists?}
    check_pr -->|Yes| use_pr[Use feature branch]
    check_pr -->|No| check_fallback{Default fallback?}

    check_fallback -->|main| use_main[Use main branch]
    check_fallback -->|latest_stable| find_latest[Find latest stable tag]
    check_fallback -->|auto_create| create_pr[Trigger auto-create PR]
    check_fallback -->|previous_compatible| find_prev[Find previous compatible build]

    use_pr --> record[Record strategy=matched]
    use_main --> record2[Record strategy=fallback_main]
    find_latest --> record3[Record strategy=fallback_latest_stable]
    create_pr --> record4[Record strategy=fallback_auto_create]
    find_prev --> record5[Record strategy=fallback_previous_compatible]
```
