---
description: Create a Jira issue ticket following the dev team's standardized template for Stories, Tasks, or Bugs
---

# Jira Issue Creation Workflow

Follow these steps to create a well-structured Jira issue ticket:

## 1. Issue Metadata
- **Issue Type**: Choose `Story`, `Task`, or `Bug`
- **Summary**: Use format `[Area/Module] – Action or Outcome`
  - Example: `Auth – Add Discord login option`

## 2. Description Section

### Background
Answer these questions:
- Why is this needed?
- What problem does it solve?
- Is this part of a larger epic or initiative?

### Goal
- What should be achieved when this ticket is completed?

### Scope
**In Scope** – List exactly what is included:
- Feature A
- Feature B

**Out of Scope** – Clearly state exclusions to prevent scope creep:
- Item X (not part of this ticket)
- Item Y (future enhancement)

## 3. Requirements

### Functional Requirements
- System must …
- User should be able to …
- API should return …

### Non-Functional Requirements
- Performance constraints
- Security considerations
- Backward compatibility

## 4. Acceptance Criteria
Use Gherkin format or bullet points:

```gherkin
GIVEN a user is not logged in
WHEN the user clicks "Login with Discord"
THEN the user is authenticated and redirected to the dashboard
```

OR bullet format:
- [ ] User can log in using Discord
- [ ] Existing local login continues to work
- [ ] Errors are handled gracefully

## 5. Technical Notes
Document implementation details:
- **APIs**: List endpoints to create/update (e.g., `POST /auth/discord`)
- **Database changes**: New tables/columns (e.g., `discord_user_id`)
- **Libraries/Tools**: Required packages (e.g., AuthJS, Discord OAuth)
- **Dependencies**: Other tickets or services required

## 6. UI/UX Notes (if applicable)
- **Screens affected**: List affected pages/components
- **Design reference**: Link to Figma/screenshots
- **UX behavior**: Loading states, error messages, animations

## 7. Edge Cases & Error Handling
Document potential issues:
- What happens if [service] fails?
- What if [data] already exists?
- Rate limiting / retry logic?

## 8. Testing Notes (optional)
- **Unit tests required**: Yes / No
- **Manual test steps**:
  1. Navigate to [page]
  2. Perform [action]
  3. Verify [result]
- **Environments**: Dev / Staging / Prod

## 9. Dependencies & Blockers
- **Blocked by**: Ticket ID (e.g., `AUTH-12`)
- **Depends on**: Other tickets/services

## 10. Definition of Done (DoD)
- [ ] Code implemented
- [ ] Tests added/updated
- [ ] Code reviewed
- [ ] QA passed
- [ ] Deployed to staging
- [ ] Documentation updated

## 11. Additional Fields
- **Priority**: Low / Medium / High / Critical
- **Labels**: `backend`, `frontend`, `auth`, `tech-debt`, `bug`, etc.
- **Assignee**: Developer name
- **Estimated Effort**: Story Points (3/5/8) or Time (~1.5 days)

---

## Quick Template

```markdown
## Summary
[Area] – [Action/Outcome]

## Background
[Why this is needed]

## Goal
[Expected outcome]

## Scope
**In Scope:**
- Item 1
- Item 2

**Out of Scope:**
- Item A

## Requirements
**Functional:**
- Requirement 1

**Non-Functional:**
- Requirement 1

## Acceptance Criteria
- [ ] Criteria 1
- [ ] Criteria 2

## Technical Notes
- APIs: 
- DB Changes: 
- Libraries: 

## Edge Cases
- Case 1
- Case 2

## DoD
- [ ] Code implemented
- [ ] Tests added
- [ ] QA passed
```
