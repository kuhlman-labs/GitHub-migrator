# Enterprise Live Migrations (ELM)

GitHub Enterprise Live Migrations move a repository from a GitHub Enterprise
Server appliance to GHE.com with near-zero downtime: the appliance backfills the
repository while the source stays **writable**, and a short operator-triggered
cutover flips it to the destination. This document is the contract for how the
migrator uses ELM, and — just as importantly — what it deliberately does **not**
do.

ELM is in public preview. Its CLI surface may change; see
[Preview risk](#preview-risk) below for how that surfaces here.

---

## The migration route contract

**The selected route lives in exactly one place**: the nullable
`repositories.migration_route` column, surfaced as
`models.Repository.MigrationRoute *string`. It sits on the main `repositories`
table beside `status`, so the strategy reader sees it without a `Preload`.

| | |
| --- | --- |
| **Legal values** | `models.MigrationRouteGEI` (`"gei"`) and `models.MigrationRouteELM` (`"elm"`) — nothing else |
| **Default** | A NULL or empty column reads as `"gei"` via `Repository.GetMigrationRoute()` |
| **Writers** | Exactly two (below) |
| **Reader** | Exactly one: `ELMStrategy.SupportsRepository` via `Repository.IsELMRouted()` |

**The default is the load-bearing part.** An unrouted repository is GEI-routed, so
no data backfill is needed and no existing repository changes behavior. There is
no separate "recommendation" field, no derived route, and no in-memory-only
selection anywhere in the codebase.

### The two writers

1. **Discovery (automatic, off by default).** When
   `discovery.Analyzer.ELMEnabled` is true and a repository exceeds the 40 GiB
   limit, the analyzer records `migration_route = "elm"` — but only for a
   GHES-sourced repository with no Azure DevOps project. It deliberately does
   **not** force `remediation_required` in that case, because the repository is
   migratable after all; the oversized flag and size details stay set so the size
   is still surfaced to the operator.

   With `ELMEnabled` false — **the default** — an oversized repository is handled
   byte-for-byte as it always has been: `remediation_required`, route left NULL.

2. **The operator (deliberate).**
   `POST /api/v1/repositories/{fullName}/migration-route` with body
   `{"route": "elm"}`. Only the two legal values are accepted; anything else is
   refused with `400` and a `reason` of `invalid_migration_route`, and the stored
   route is left unchanged. A `null` route clears the column back to the GEI
   default.

### The one reader

`ELMStrategy.SupportsRepository(repo)` returns true **if and only if**
`repo.IsELMRouted()` **and** the repository has no ADO project **and** its source
is GHES.

`ELMStrategy` is registered **first** in `NewStrategyRegistry`, because
`GetStrategy` returns the first match and `GitHubMigrationStrategy` matches every
non-ADO repository. Registering it last would make the feature permanently
unreachable. The route check inside `SupportsRepository` is what makes ELM-first
registration safe: an unrouted repository falls straight through to GEI.

---

## Corridor restriction: GHES → GHE.com only

ELM exists for **data residency migrations from a GHES appliance to GHE.com**. It
is not a general GEI replacement:

- A **github.com / GHEC** source is never ELM-routed.
- An **Azure DevOps** repository is never ELM-routed, even if something writes
  `"elm"` into its route column — `SupportsRepository` re-checks the corridor
  independently of the route.

Everything outside that corridor keeps using the existing GEI (`GitHub`) and
Azure DevOps strategies, unchanged.

---

## Scope: repositories only

**ELM carries repositories. It does not carry organization settings, teams, or
projects.**

Those remain the job of the existing separate passes
(`internal/migration/team_executor.go` and the project passes), which must be run
over the ELM-migrated set exactly as they are for GEI-migrated repositories. A
repository reaching `migration_complete` via ELM is **not** a signal that its org
membership, team permissions, or projects have moved.

---

## SSH security posture

The ELM exporter is only reachable from the appliance itself: `API_URL` is fixed
at `http://localhost:1738` and the HMAC key is readable only through `ghe-config`
in the administrative shell. The migrator therefore drives ELM over SSH into that
shell.

- **Host-key verification is mandatory.** The client's `HostKeyCallback` is built
  with `knownhosts.New(KnownHostsPath)`. There is no `ssh.InsecureIgnoreHostKey`
  on any code path, no debug flag, and no test-only escape hatch.
- **Construction fails closed.** An empty or unreadable `known_hosts` path, or an
  empty, unreadable or undecryptable private key, returns an error and a nil
  transport. There is no degraded fallback.
- **The key is never logged.** The private key material and its passphrase never
  reach a log record, an error message, or a command string.
- **The HMAC key never transits this process.** It is resolved inside the admin
  shell via `$(ghe-config secrets.elm-exporter.elm-exporter-hmac-keys)`.
- **Every interpolated value is shell-quoted.** Org, repo, target org, target
  repo, target API URL, visibility, migration id and page cursor are all
  single-quoted with embedded quotes escaped before they reach the admin shell.
- Use a **dedicated site-admin key** for this connection, not a shared operator
  key.

---

## Cutover is manual, and `--force` is unreachable

Cutover is the single **irreversible, downtime-causing** step of a live
migration. It is therefore:

- **Operator-triggered only** — `POST /api/v1/repositories/{fullName}/cutover`,
  behind a confirmation dialog in the dashboard. Nothing transitions into cutover
  automatically; the poll loop advances a migration to `awaiting_cutover` and
  stops there, for as long as it takes.
- **Gated at two independent layers.** The HTTP handler refuses with `409` and a
  machine-readable `reason` (`elm_not_ready` / `elm_no_record`) unless the
  persisted record reports readiness, dispatching no command. `ELMService.Cutover`
  then re-checks the persisted record *and* re-confirms with a fresh status call
  before issuing anything.
- **Never forced.** `elm.Client.CutoverToDestination` takes no force parameter and
  the command builder structurally cannot emit `--force` for the cutover verb.
  There is no API field, config key, or UI control in this tool that can reach it.

---

## Dry run is preflight-only

ELM's documented CLI surface has no preflight verb, so an **ELM dry run creates no
migration**. It issues `elm migration list` to prove the appliance is reachable
and the credentials work, runs the repository eligibility checks, and records
`dry_run_complete` (or `dry_run_failed`). It never issues `migration create`,
`migration start`, or `cutover-to-destination`, and the source is never locked.

---

## Concurrency ceilings

Two ceilings bound how many live migrations may be in flight. Both are
configurable (`elm.max_concurrent_source`, `elm.max_concurrent_destination`), so a
change in GitHub's published limits is a config edit rather than a release. When a
ceiling is reached the repository is left **queued** and retried later, not failed.

**Source ceiling — 10 concurrent per source instance.** Counted by joining
`elm_migrations` to `repositories` and grouping on the existing indexed
`repositories.source_id` foreign key. Repositories with a NULL `source_id` are not
dropped from the count; they are all attributed to one well-defined bucket
(`storage.ELMUnknownSourceBucket`), so the ceiling still applies to them
collectively.

**Destination ceiling — 20 concurrent, per deployment.** This one is a **global**
count, not a grouped one, and that is deliberate: `config.Destination` is a single
`DestinationConfig` with one `base_url`, so a running instance of this tool targets
**exactly one destination**. Per-deployment and per-destination-enterprise are
therefore the same number. `DestinationURL` / `DestinationFullName` on a repository
are nullable free-text strings with no foreign key and are **not** parsed to
synthesise an enterprise key.

> **If multi-destination support ever lands, this count must become grouped.** It
> is a global count today only because there is exactly one destination.

---

## Lifecycle and the worker

```
queued_for_migration
   -> syncing            (elm migration create + start; ELM migration id persisted)
   -> awaiting_cutover   (appliance reports the backfill ready — may sit here for days)
   -> cutting_over       (operator triggers cutover)
   -> migration_complete
```

The poll loop that advances these states runs **outside** `MigrationWorker`,
started from `cmd/server/main.go` beside the existing worker. A repository sitting
in `awaiting_cutover` occupies **no worker slot**, so a long-running backfill
cannot starve GEI migrations.

On startup the service pages `elm migration list` and cross-references it against
the persisted `elm_migrations` rows: migrations still in flight across a restart
are re-adopted, and a persisted id the appliance no longer knows about is marked
failed with a reason.

---

## Rollback and the post-cutover recovery runbook

**Before cutover**, rollback is clean: `elm migration cancel` stops the backfill
with no impact on the source repository, which stayed writable throughout.

**After cutover, there is no automated rollback.** ELM **archives the source
repository** at cutover. This tool deliberately does not automate recovery; it
records the handles an operator needs and stops there.

Recovery runbook:

1. Read the persisted ELM migration id and the pre-cutover state from the
   repository's ELM record — `GET /api/v1/repositories/{fullName}/elm`.
2. On the GHES appliance, **unarchive the source repository**.
3. Redirect clients back to the source (remotes, webhooks, CI configuration).
4. Delete or rename the destination repository on GHE.com if it must not remain.
5. Clear the route (`{"route": null}`) if the repository should go back to the GEI
   corridor, or leave it as `"elm"` and re-run once the cause is fixed.

Steps 2–4 are manual by design: they touch a live source of truth and a
destination that may already have received writes.

---

## Configuration

All keys are under `elm.` in the config file, or `GHMIG_ELM_*` in the environment.
ELM is inert unless `elm.enabled` is true, and an enabled-but-incomplete config is
refused at startup rather than half-started.

| Key | Env | Notes |
| --- | --- | --- |
| `elm.enabled` | `GHMIG_ELM_ENABLED` | Master switch, default `false` |
| `elm.ssh_host` | `GHMIG_ELM_SSH_HOST` | GHES appliance |
| `elm.ssh_port` | `GHMIG_ELM_SSH_PORT` | Default 22; GHES admin shell is normally 122 |
| `elm.ssh_user` | `GHMIG_ELM_SSH_USER` | Dedicated site-admin user |
| `elm.ssh_private_key_path` | `GHMIG_ELM_SSH_PRIVATE_KEY_PATH` | Required; never logged |
| `elm.ssh_private_key_passphrase` | `GHMIG_ELM_SSH_PRIVATE_KEY_PASSPHRASE` | Optional; never logged |
| `elm.ssh_known_hosts_path` | `GHMIG_ELM_SSH_KNOWN_HOSTS_PATH` | **Required** — host-key verification is mandatory |
| `elm.target_api_url` | `GHMIG_ELM_TARGET_API_URL` | Destination (GHE.com) API endpoint |
| `elm.pat_name` | `GHMIG_ELM_PAT_NAME` | Named appliance credential (preview documents `system-pat`) |
| `elm.poll_interval_seconds` | `GHMIG_ELM_POLL_INTERVAL_SECONDS` | Backfill poll cadence |
| `elm.max_concurrent_source` | `GHMIG_ELM_MAX_CONCURRENT_SOURCE` | Default 10, per source instance |
| `elm.max_concurrent_destination` | `GHMIG_ELM_MAX_CONCURRENT_DESTINATION` | Default 20, **per deployment** |

---

## Preview risk

ELM is in public preview, so its CLI output format may drift. Two controls make a
drift visible rather than silently wrong:

- The client's command builders are pinned by tests asserting **exact command
  strings**.
- Unrecognised output raises a **typed parse error** and leaves `cutover_ready`
  **unchanged** rather than defaulting it. A drifted status output reads as a
  stalled sync that needs a human, never as "ready to cut over".

Appliance behavior itself is **operator-validated, not CI-validated**: no
automated test in this repository can reach a GHES appliance over SSH or a GHE.com
data-residency enterprise. The seam-level proof is the end-to-end test driving a
fake `elm.Transport` through create → start → status → cutover.

---

## API reference

See [API.md](API.md) for the full request/response shapes:

- `POST /api/v1/repositories/{fullName}/migration-route` — set or clear the route
- `POST /api/v1/repositories/{fullName}/cutover` — operator-triggered cutover
- `GET /api/v1/repositories/{fullName}/elm` — live-migration status
