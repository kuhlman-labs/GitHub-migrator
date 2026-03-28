---
name: pre-flight
description: Run pre-flight checks before starting a migration batch
user_invocable: true
---

# Pre-Flight Check

You are running pre-flight checks to determine if a migration batch is ready to proceed.

## Workflow

1. **Check overall readiness**: Call `get_migration_readiness` to get the readiness score and component breakdown. The readiness score (0-100%) indicates how prepared the migration is.

2. **User mapping status**: Call `get_user_mapping_stats` to check user mapping completeness. Report:
   - Total users found in source repositories
   - Users with confirmed mappings vs unmapped
   - Mapping completion percentage

3. **Team mapping status**: Call `get_team_mapping_stats` to check team mapping completeness. Report:
   - Total teams found
   - Teams with confirmed mappings vs unmapped
   - Mapping completion percentage

4. **Repository blockers**: Call `get_repository_blockers` to identify any repositories with blockers that would prevent migration. List each blocker with its severity.

5. **Present go/no-go checklist**: Format results as a clear checklist:

   ```
   Pre-Flight Checklist
   --------------------
   [PASS/FAIL] User mappings: XX% complete (Y unmapped)
   [PASS/FAIL] Team mappings: XX% complete (Y unmapped)
   [PASS/FAIL] No critical blockers (X blockers found)
   [PASS/FAIL] Dry runs completed for batch repos

   Overall: READY / NOT READY (Score: XX%)
   ```

6. **Recommendation**: Based on results:
   - **READY** (score >= 80%, no critical blockers): "Safe to proceed. Consider running a dry-run first if not already done."
   - **CAUTION** (score 50-79%): "Some issues to address. List specific items to fix before proceeding."
   - **NOT READY** (score < 50%): "Significant preparation needed. Prioritize the following items..."

## If Not Ready

Suggest specific remediation steps:
- For unmapped users: run `suggest_user_mappings` to auto-generate suggestions
- For unmapped teams: run `suggest_team_mappings` to auto-generate suggestions
- For blockers: list each blocker with its resolution guidance
- For missing dry runs: suggest scheduling dry-run batches first
