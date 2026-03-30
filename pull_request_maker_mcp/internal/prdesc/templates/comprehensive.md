Generate a PR title and description using the rules below. Base everything on the actual code changes in the diff.

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
<!-- Always include the Jira ticket link for traceability -->
[{ticket_id}](https://mekari.atlassian.net/browse/{ticket_id})

### Summary
<!-- Provide a brief summary of the changes (2-3 sentences). What does this PR do? -->

### Motivation
<!-- Why are these changes needed? What problem does it solve? -->

### Changes
<!-- List every meaningful change. Be specific — what was added, modified, or removed. -->
-

### Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Refactoring (no functional changes)
- [ ] Documentation update
- [ ] Configuration change

### Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

### File Changes
<!-- List of files changed with a brief description of the change -->
-

### Checklist
- [ ] My code follows the project's coding standards
- [ ] I have performed a self-review of my code
- [ ] I have added/updated tests that prove my fix/feature works
- [ ] New and existing tests pass locally
- [ ] I have updated relevant documentation

### Dev Self Test
<!-- Add screenshots or logs to demonstrate the changes work as expected -->

### AI Prompt
<!-- Describe the prompt used to generate this PR (attach image if applicable) -->
-
