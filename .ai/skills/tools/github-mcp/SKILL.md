# GitHub MCP Skill

**Tech Stack:** GitHub MCP Server (Official)  
**Purpose:** GitHub API integration, repository management, issue tracking, PR workflows  
**Version:** MCP Protocol (2024+)

---

## Architecture Overview

**GitHub MCP Server** provides direct access to GitHub's API through the Model Context Protocol. It enables:

- **Repository Management:** Create repos, manage files, branches, tags
- **Issue Tracking:** Create, search, update issues with full metadata
- **Pull Request Workflows:** Create PRs, request reviews, merge, update branches
- **Code Search:** Search code, issues, PRs, repositories, users across GitHub
- **Team Collaboration:** Manage teams, request reviews, comment on PRs

**When to Use:**
- Automating issue creation from errors/bugs
- Creating PRs programmatically
- Searching for solutions in GitHub issues/discussions
- Managing repository files without git CLI
- Workflow automation (CI/CD triggers, status checks)

---

## Architecture Patterns

### 1. Prefer MCP Tools Over gh CLI

**Pattern:** Use GitHub MCP tools as primary method, fall back to `gh` CLI only if MCP tool unavailable.

**Rationale:**
- Direct API integration (type-safe, no shell parsing)
- Better error handling (structured responses)
- Faster execution (no subprocess overhead)

**Example Decision:**
```
Need to create issue?
  ├─ MCP Tool available? → Use mcp_github_issue_write
  └─ MCP Tool missing? → Fall back to gh issue create
```

### 2. Search Before Create

**Pattern:** Always search existing issues/PRs before creating new ones.

```
1. Search issues with relevant keywords
2. Check if issue already exists
3. If exists → Comment on existing issue
4. If not → Create new issue
```

**Benefits:**
- Avoid duplicates
- Discover existing solutions
- Better issue organization

### 3. Atomic PR Workflow

**Pattern:** One logical change per PR, proper metadata from creation.

```
1. Create branch
2. Make changes (push_files for multiple files)
3. Create PR with: title, body, labels, reviewers
4. Request reviews (if needed)
5. Merge after approval
```

**Anti-Pattern:** ❌ Creating PR first, adding metadata later (incomplete context).

---

## Best Practices

1. **Use MCP Tools First**
   - Check available MCP tools before using `gh` CLI
   - MCP tools are type-safe and provide better error messages
   - Fall back to CLI only if specific MCP tool doesn't exist

2. **Search Effectively**
   - Use specific search tools: `search_issues`, `search_code`, `search_pull_requests`
   - Combine filters: `label:bug state:open repo:owner/name`
   - Use pagination for large result sets

3. **Structured Issue/PR Bodies**
   - Use markdown headings for sections
   - Include context, reproduction steps, expected behavior
   - Reference related issues/PRs with `#123` syntax
   - Add labels for categorization

4. **Batch File Operations**
   - Use `push_files` for multiple file changes in single commit
   - Avoid multiple commits for related changes
   - Group logically related file operations

5. **Proper Branch Management**
   - Create branches from latest `main` before changes
   - Use descriptive branch names: `feat/user-auth`, `fix/login-error`
   - Clean up branches after PR merge

6. **Review Workflow**
   - Use `request_copilot_review` for automated feedback
   - Add human reviewers with `update_pull_request`
   - Address review comments before merge

7. **Error Handling**
   - Check MCP tool responses for errors
   - Handle rate limits gracefully
   - Validate inputs before API calls

---

## Common Patterns

### Pattern 1: Automated Issue Creation from Errors

**Scenario:** Application error detected, create GitHub issue automatically.

```txt
1. Extract error details (message, stack trace, context)
2. Search existing issues for similar error
3. If not found:
   - Format issue body with error details
   - Add labels: bug, automated
   - Create issue with mcp_github_issue_write
4. If found:
   - Add comment to existing issue with new occurrence
```

**Key Points:**
- Avoid duplicate issues (search first)
- Include reproduction steps
- Add relevant labels for triage

### Pattern 2: Multi-File PR Creation

**Scenario:** Implement feature across multiple files, create PR.

```txt
1. Create branch: mcp_github_create_branch
2. Prepare file changes (array of {path, content})
3. Push all files: mcp_github_push_files
4. Create PR: mcp_github_create_pull_request
   - Title: "feat: user authentication"
   - Body: Overview, changes, testing notes
   - Labels: feature
   - Reviewers: team members
```

**Key Points:**
- Use `push_files` for atomic multi-file commits
- Include complete PR metadata at creation
- Reference related issues with "Closes #123"

### Pattern 3: Research Solutions via GitHub Search

**Scenario:** Encountering error, search for solutions in GitHub issues.

```txt
1. Extract error message keywords
2. Search issues: mcp_github_search_issues
   - Query: "error message language:javascript"
   - Sort by: reactions (most helpful solutions)
3. Filter results: closed issues with accepted answers
4. Review top 3-5 issues for solution patterns
5. Apply solution to codebase
```

**Key Points:**
- Search across all repositories (not just one repo)
- Prioritize issues with high reactions/comments
- Check closed issues (likely have solutions)

### Pattern 4: PR Review & Merge Workflow

**Scenario:** PR created, need review and merge.

```txt
1. Request Copilot review: mcp_github_request_copilot_review
2. Review automated feedback
3. Fix issues identified by review
4. Update PR branch: mcp_github_update_pull_request_branch
5. Request human review: mcp_github_update_pull_request (add reviewers)
6. After approval: mcp_github_merge_pull_request
```

**Key Points:**
- Use Copilot review for quick automated feedback
- Update branch before merge (avoid conflicts)
- Use squash merge for clean history

### Pattern 5: Issue Triage & Management

**Scenario:** Manage open issues, categorize, assign.

```txt
1. List open issues: mcp_github_list_issues (state: OPEN)
2. For each issue:
   - Read issue details
   - Determine category (bug, feature, question)
   - Add labels: mcp_github_issue_write (update)
   - Assign to team member if actionable
3. Close stale issues (no activity > 30 days)
```

**Key Points:**
- Use labels for categorization
- Assign issues for accountability
- Close or archive stale issues

---

## Anti-Patterns

### ❌ Using gh CLI When MCP Tool Exists
**Why:** Slower, no type safety, harder error handling.  
**Instead:** Check MCP tools first, use `mcp_github_*` functions.

### ❌ Creating PRs Without Metadata
**Why:** Incomplete context, requires follow-up edits.  
**Instead:** Include title, body, labels, reviewers at creation time.

### ❌ Skipping Search Before Creating Issues
**Why:** Duplicate issues, fragmented discussions.  
**Instead:** Always search with `search_issues` before creating.

### ❌ Multiple Single-File Commits
**Why:** Cluttered history, hard to review, inefficient.  
**Instead:** Use `push_files` with array of file changes for atomic commits.

### ❌ Ignoring Rate Limits
**Why:** API failures, blocked operations.  
**Instead:** Batch operations, check rate limit status, handle 429 errors.

---

## Integration with Development Workflow

### With Issue Management
**Workflow:**
```
Error Detected → Search Issues → Create/Comment → Fix → Close Issue
```

**Tools:**
- `search_issues` → Find existing reports
- `issue_write` → Create new issue
- `add_issue_comment` → Update existing issue

### With PR Workflow
**Workflow:**
```
Feature → Branch → Changes → PR → Review → Merge → Cleanup
```

**Tools:**
- `create_branch` → Start work
- `push_files` → Commit changes
- `create_pull_request` → Open PR
- `request_copilot_review` → Automated review
- `merge_pull_request` → Complete PR

### With Code Search
**Workflow:**
```
Problem → Search Code/Issues → Find Solution → Apply → Document
```

**Tools:**
- `search_code` → Find implementation examples
- `search_issues` → Find discussions/solutions
- `search_repositories` → Discover relevant projects

---

## Performance Considerations

1. **Pagination Strategy**
   - Default: 30 items per page
   - Max: 100 items per page
   - Use pagination for large result sets
   - Avoid fetching all results if only need top N

2. **Rate Limits**
   - Authenticated: 5000 requests/hour
   - Search API: 30 requests/minute
   - Monitor `X-RateLimit-Remaining` header
   - Batch operations when possible

3. **Search Optimization**
   - Use specific queries (avoid broad searches)
   - Filter by repository/organization
   - Limit results with pagination
   - Cache search results when appropriate

---

## Security Guidelines

1. **Token Management**
   - MCP server handles authentication (no manual tokens)
   - Never expose GitHub tokens in code/logs
   - Use minimal required scopes

2. **Repository Access**
   - Respect repository permissions
   - Don't expose private repository data
   - Validate user permissions before operations

3. **Sensitive Data**
   - Don't include secrets in issue/PR descriptions
   - Sanitize error messages before creating issues
   - Use GitHub Secrets for CI/CD credentials

---

## Quick Reference

| Operation | MCP Tool | Use Case |
|-----------|----------|----------|
| Create issue | `mcp_github_issue_write` | Bug reports, feature requests |
| Search issues | `mcp_github_search_issues` | Find existing issues/solutions |
| Create PR | `mcp_github_create_pull_request` | Submit code changes |
| Push files | `mcp_github_push_files` | Multi-file atomic commit |
| Search code | `mcp_github_search_code` | Find implementation examples |
| Request review | `mcp_github_request_copilot_review` | Automated code review |
| Merge PR | `mcp_github_merge_pull_request` | Complete pull request |
| Create branch | `mcp_github_create_branch` | Start new work |
| List issues | `mcp_github_list_issues` | Issue management |
| Update PR | `mcp_github_update_pull_request` | Add reviewers, update metadata |

---

## Tool Categories (Activation Required)

Some GitHub MCP tools require activation via `activate_*` functions:

- `activate_github_issue_management` → Issue CRUD, comments, sub-issues
- `activate_github_pull_request_management` → PR CRUD, reviews, merge
- `activate_github_repository_management` → Files, branches, tags
- `activate_github_workflow_management` → GitHub Actions
- `activate_github_search_tools` → Code/Issue/PR/Repo/User search
- `activate_github_security_management` → Security alerts

**Pattern:** Activate category when first needed, reuse for subsequent operations.

---

## Common Debugging Scenarios

### Scenario: Issue Creation Fails
**Steps:**
1. Check error message from MCP tool response
2. Validate required fields (owner, repo, title, body)
3. Verify repository access permissions
4. Check rate limit status

### Scenario: PR Merge Blocked
**Steps:**
1. Check PR status: `mcp_github_pull_request_read` (method: get_status)
2. Verify required checks passed
3. Ensure no merge conflicts
4. Update branch if needed: `update_pull_request_branch`

### Scenario: Search Returns No Results
**Steps:**
1. Verify search query syntax
2. Check repository/organization filters
3. Try broader search terms
4. Confirm issues/code exists in target scope

---

**Last Updated:** 2025-11-04  
**Maintained By:** skill-creator agent
