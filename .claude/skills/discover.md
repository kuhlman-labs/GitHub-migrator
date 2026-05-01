---
name: discover
description: Discover repositories from a configured migration source
user_invocable: true
---

# Discover Repositories

You are helping the user discover repositories from a configured GitHub migration source.

## Workflow

1. **List available sources**: Call the `list_sources` MCP tool to show all configured migration sources. Present them in a table with ID, name, type, base URL, repository count, and last sync time.

2. **Select source**: If the user specified a source (by name or ID), use that. Otherwise, ask which source to discover from.

3. **Start discovery**: Call the `start_discovery` MCP tool with the selected `source_id`. Inform the user that discovery has started.

4. **Monitor progress**: Poll `get_discovery_progress` every 10 seconds until the status is no longer "in_progress". Show progress updates as they come in (e.g., "Discovered 45/120 repositories...").

5. **Analyze results**: Once discovery is complete, call `analyze_repositories` with the source's organization to summarize what was found. Include:
   - Total repositories discovered
   - Breakdown by complexity (simple, moderate, complex)
   - Languages and sizes

6. **Recommend next steps**: Based on the results, suggest:
   - Running `/plan-migration` to create a migration plan
   - Running `/pre-flight` to check migration readiness
   - Reviewing any large or complex repositories that may need special attention

## Error Handling

- If discovery fails, show the error and suggest checking source credentials
- If no sources exist, guide the user to configure a source first via the API
- If a discovery is already running, show its progress instead of starting a new one
