# Project Development Guidelines

**Agent-Primary Model:** Agents execute ALL tasks. Skills provide tech-stack reference that agents use.

---

## CRITICAL: Agent Orchestration - Wann welchen Agent verwenden

**RULE: Für JEDE Aufgabe gibt es einen passenden Agent oder Agent-Chain!**

### Single Agent Tasks

| User Request | Agent | Beispiel |
|--------------|-------|----------|
| **Feature Implementation** | `feature-implementer` | "Implement user registration with email verification" |
| **Test Coverage Improvement** | `test-implementer` | "Increase test coverage for handlers package" |
| **Code Refactoring** | `code-refactor-master` | "Reorganize the components folder structure" |
| **Architecture Review** | `code-architecture-reviewer` | "Review the authentication module architecture" |
| **Documentation** | `documentation-architect` | "Create API documentation for user endpoints" |
| **Auth Debugging** | `auth-route-debugger` | "Getting 401 error on /api/users" |
| **Frontend Debugging** | `frontend-error-fixer` | "React component not rendering correctly" |
| **TypeScript Errors** | `auto-error-resolver` | "Fix all TypeScript compilation errors" |
| **Auth Testing** | `auth-route-tester` | "Test the authenticated endpoints" |
| **Research** | `web-research-specialist` | "Find best practices for handling file uploads" |
| **Skill Creation** | `skill-creator` | "Create skills for React and Express" / "Need skill reference for MUI v7" |

### Agent Chains (Multi-Step Tasks)

**Complex Implementation Chain:**

```txt
User: "Implement complete user registration system"

Step 1: web-research-specialist
→ Research best practices for registration flows
→ Find security considerations
→ Identify proven patterns

Step 2: plan-reviewer  
→ Review the implementation plan
→ Identify potential issues
→ Validate architectural decisions

Step 3: feature-implementer
→ Implement database schema
→ Create backend services
→ Build frontend components
→ Integrate all layers
```

**Refactoring Chain:**

```txt
User: "Refactor the authentication module"

Step 1: refactor-planner
→ Analyze current structure
→ Identify problems
→ Create refactoring strategy

Step 2: code-refactor-master
→ Execute the refactoring plan
→ Move files systematically
→ Update all imports
→ Verify nothing breaks
```

**Research → Implementation Chain:**

```txt
User: "Add real-time notifications (not sure how)"

Step 1: web-research-specialist
→ Research WebSocket vs SSE vs Polling
→ Find implementation examples
→ Compare trade-offs

Step 2: plan-reviewer
→ Review chosen approach
→ Validate architecture fit

Step 3: feature-implementer
→ Implement chosen solution
```

### Decision Tree

```
User Request
    │
    ├─ New Feature? 
    │   ├─ Simple/Clear → feature-implementer
    │   └─ Complex/Unclear → web-research → plan-reviewer → feature-implementer
    │
    ├─ Refactoring?
    │   ├─ Clear scope → code-refactor-master
    │   └─ Needs planning → refactor-planner → code-refactor-master
    │
    ├─ Debugging?
    │   ├─ Auth issue → auth-route-debugger
    │   ├─ Frontend issue → frontend-error-fixer
    │   └─ TypeScript errors → auto-error-resolver
    │
    ├─ Testing?
    │   ├─ Route testing → auth-route-tester
    │   └─ Coverage improvement → test-implementer
    │
    ├─ Review/Documentation?
    │   ├─ Architecture → code-architecture-reviewer
    │   ├─ Plan review → plan-reviewer
    │   └─ Documentation → documentation-architect
    │
    └─ Research needed?
        └─ web-research-specialist → (plan-reviewer) → appropriate agent
```

### Wann welche Chain?

**Einfach → Single Agent:**

- Klare Anforderungen
- Bekannter Scope
- Standard-Patterns
- Beispiel: "Add validation to login form"

**Komplex → Chain:**

- Unklare Anforderungen
- Neue Technologie
- Große Scope
- Beispiel: "Implement real-time collaboration"

**Skills' Role:**
Agents verwenden Skills als **Referenz-Material**:

- Express-Patterns → Wie schreibe ich Express-Code?
- React-Patterns → Welche React-Patterns nutzen?
- Git-Workflow → Wie formatiere ich Commits?

---

## Available Agents (Load from .ai/agents/)

**Implementation:**

- feature-implementer → Implement new features across all layers (database → backend → frontend)

**Testing:**

- test-implementer → Systematically increase test coverage (unit, integration, benchmarks)
- auth-route-tester → Test authenticated endpoints functionality

**Debugging:**

- auth-route-debugger → Auth errors, 401/403, JWT issues
- frontend-error-fixer → React errors, UI bugs
- auto-error-resolver → TypeScript compilation errors

**Refactoring:**

- refactor-planner → Plan refactoring strategy
- code-refactor-master → Execute refactorings

**Review:**

- code-architecture-reviewer → Architecture reviews
- plan-reviewer → Review implementation plans

**Documentation:**

- documentation-architect → Create comprehensive docs

**Research:**

- web-research-specialist → Research solutions

**Setup & Skills:**

- skill-creator → Create tech-stack skills (Express, React, MUI, etc.)

**Load agents with:** `@workspace load .ai/agents/[agent-name].md`

---

## Available Skills (Reference Material for Agents)

**Backend Skills (Tech-Stack Reference):**

- fiber → Fiber web framework patterns, handler structure, middleware
- service-layer-patterns → Service Layer best practices (Constructor Injection, Caching, ESI Integration, Graceful Degradation)

**Frontend Skills (Tech-Stack Reference):**

- nextjs → Next.js App Router, Server/Client Components
- radix-ui → Radix UI components, controlled patterns

**Database Skills (Tech-Stack Reference):**

- postgresql → PostgreSQL patterns, connection pooling
- redis → Redis caching, key naming conventions
- sqlite → SQLite read-only patterns (SDE integration)

**Testing Skills (Tech-Stack Reference):**

- playwright → E2E testing, Page Object Model

**Generic Skills (Methodology Reference):**

- git-workflow → Git Flow, GitHub Flow specifications
- documentation-standards → JSDoc, API docs formats

**Skills provide syntax and patterns that agents reference during execution.**

---

## Architecture Decision Records (ADRs)

**Location:** `docs/adr/`

**CRITICAL:** Before making architectural changes, ALWAYS check existing ADRs!

**Available ADRs:**
- ADR-001: Tech Stack (Go + Next.js + PostgreSQL + Redis)
- ADR-004: Frontend OAuth PKCE Flow
- ADR-009: Shared Redis Infrastructure
- ADR-010: SDE Database Path Convention
- ADR-011: Worker Pool Pattern
- ADR-012: Redis Caching Strategy (Market Orders + Character Data)
- ADR-013: Timeout Handling & Partial Content
- ADR-014: ESI Integration Pattern (eve-esi-client Usage)

**When to consult ADRs:**
- ✅ Before adding new dependencies
- ✅ Before changing database schemas
- ✅ Before modifying auth flows
- ✅ Before implementing caching logic
- ✅ Before refactoring core architecture

**ADR Template:** `docs/adr/000-template.md`

**Process:**
1. Check if ADR exists for your change area
2. If exists → Follow it (or propose supersession with new ADR)
3. If not exists → Create new ADR for significant decisions

---

## Backend Development

**Skill Location:** `.ai/skills/generic/git-workflow/SKILL.md`

Git conventions, commit message formats, branch naming, PR workflow.

**To create:** Ask "Create Git workflow skill"  
**To load:** `@workspace .ai/skills/generic/git-workflow/SKILL.md`

---

## PLACEHOLDER: Documentation Standards

**Skill Location:** `.ai/skills/generic/documentation-standards/SKILL.md`

JSDoc, TSDoc, API documentation formats, README structures.

**To create:** Ask "Create documentation standards skill"  
**To load:** `@workspace .ai/skills/generic/documentation-standards/SKILL.md`

---

## Backend Development

**Skill Location:** `.ai/skills/backend/fiber/SKILL.md`

**Architecture:** Layered (Handler → Service → Repository → Database)

**Key Patterns:**
- Dependency injection (constructor-based)
- Context propagation with timeouts
- Structured error handling
- Repository pattern for data access

**To load:** `@workspace .ai/skills/backend/fiber/SKILL.md`

---

## Frontend Development

**Next.js:** `.ai/skills/frontend/nextjs/SKILL.md`
**Radix UI:** `.ai/skills/frontend/radix-ui/SKILL.md`

**Architecture:** App Router (Server + Client Components), React Context

**Key Patterns:**
- Server Components by default
- Client Components for interactivity (`"use client"`)
- React Context for auth state
- Controlled components (Radix UI)
- API integration with fetch

**To load:** `@workspace .ai/skills/frontend/*/SKILL.md`---

## Database & ORM

**PostgreSQL:** `.ai/skills/database/postgresql/SKILL.md`
**Redis:** `.ai/skills/database/redis/SKILL.md`
**SQLite:** `.ai/skills/database/sqlite/SKILL.md`

**Architecture:** Dual-database (PostgreSQL for dynamic data, SQLite for static SDE)

**Key Patterns:**
- Connection pooling (pgxpool)
- Repository pattern (pgx/v5)
- Batch operations for performance
- Cache-aside pattern (Redis)
- Read-only mode (SQLite SDE)

**To load:** `@workspace .ai/skills/database/*/SKILL.md`

---

## Testing

**Skill Location:** `.ai/skills/testing/playwright/SKILL.md`

**Architecture:** E2E testing with Page Object Model

**Key Patterns:**
- Accessibility-based selectors (`getByRole`, `getByLabel`)
- Auto-waiting (no manual waits)
- Page Object Model for reusability
- API mocking for deterministic tests
- Parallel execution with isolation

**To load:** `@workspace .ai/skills/testing/playwright/SKILL.md`
