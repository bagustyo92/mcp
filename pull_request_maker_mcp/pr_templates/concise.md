Generate a PR title and description using the rules below. Keep everything short and to the point.

---

## PR TITLE FORMAT (required)
Use this exact format:
```
{ticket_id}: {ticket title} - [FULL_COPILOT]
```
Examples:
- `TD-1234: Add employee attendance endpoint - [FULL_COPILOT]`
- `CORE-567: Fix payroll calculation bug - [FULL_COPILOT]`

Extract `{ticket_id}` from the source branch name (e.g. `feature/TD-1234-add-attendance` → `TD-1234`).
If no ticket ID is found, use `NO-TICKET` as the placeholder.

---

## PR DESCRIPTION

### Jira Ticket
[{ticket_id}](https://mekari.atlassian.net/browse/{ticket_id})

### Summary
<!-- 1-2 sentences max — what does this PR do? -->

### Key Changes
<!-- 3-5 bullet points only — the most important changes -->
-

### File Changes
<!-- Files changed with one-line description each -->
-

### Dev Self Test
<!-- How was this tested? Screenshot or log if applicable -->
-

### AI Prompt
<!-- Prompt used to generate this PR (attach image if applicable) -->
-
