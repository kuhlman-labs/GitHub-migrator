---
name: plan-migration
description: Create a comprehensive migration plan for discovered repositories
user_invocable: true
---

# Plan Migration

You are helping the user create a comprehensive migration plan for their repositories.

## Workflow

1. **Analyze repositories**: Call `analyze_repositories` to get an overview of all discovered repositories. If the user specified a source or organization, filter to that scope.

2. **Identify blockers**: Call `get_repository_blockers` to find repositories with migration blockers. Present any blockers clearly with their severity and recommended resolution.

3. **Check dependencies**: For repositories flagged as high complexity, call `check_dependencies` to understand inter-repository dependencies that affect migration ordering.

4. **Generate wave plan**: Call `plan_waves` to create a phased migration plan. Present each wave with:
   - Wave number and name
   - Repository count and total size
   - Estimated complexity
   - Dependencies on prior waves

5. **Find pilot candidates**: Call `find_pilot_candidates` to identify the best repositories to migrate first. These should be low-risk, low-complexity repos that validate the migration process.

6. **Present the plan**: Compile everything into a clear migration plan:
   - **Executive summary**: Total repos, estimated waves, timeline
   - **Pilot phase**: Recommended pilot repositories with rationale
   - **Wave breakdown**: Each wave with its repositories
   - **Risk register**: Blockers and complex repos requiring attention
   - **Prerequisites**: User/team mappings needed, blockers to resolve

## Recommendations

- Suggest resolving blockers before starting migration
- Recommend running user and team mapping suggestions (`suggest_user_mappings`, `suggest_team_mappings`)
- Advise running `/pre-flight` before each batch
- For large migrations (100+ repos), recommend starting with a pilot batch of 3-5 simple repositories
