---
name: migration-status
description: Get a comprehensive view of migration progress and action items
user_invocable: true
---

# Migration Status

You are providing the user with a comprehensive migration status dashboard.

## Workflow

1. **Executive summary**: Call `get_executive_summary` to get the high-level migration overview. Present:
   - Overall progress percentage
   - Repositories by status (pending, in-progress, completed, failed)
   - Migration velocity (repos/day)
   - Organization-level breakdown

2. **Action items**: Call `get_action_items` to identify what needs attention. Categorize by priority:
   - **Critical**: Failed migrations, blocked repos
   - **High**: Repos ready for migration, incomplete mappings
   - **Medium**: Pending dry runs, suggested optimizations

3. **Active migrations**: Call `get_migration_progress` for any in-progress batches to show real-time status of active work.

4. **Present dashboard**: Format everything as a clear status report:
   - Progress bar or percentage for overall migration
   - Table of active batches with their status
   - Prioritized action items list
   - Key metrics (velocity, completion ETA based on current rate)

## Follow-up Suggestions

Based on the status, suggest relevant next actions:
- If there are failed migrations: offer to show logs via `get_migration_logs` or suggest rollback
- If there are repos ready to migrate: suggest creating the next batch
- If mappings are incomplete: suggest running mapping suggestions
- If all repos are done: congratulate and suggest post-migration validation
