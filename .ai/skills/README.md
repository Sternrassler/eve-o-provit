# Skills Directory

**Purpose:** Tech-stack reference material for agents

Skills are created **on-demand** using the `skill-creator` agent. No pre-installation needed.

## How to Create Skills

Simply ask your AI assistant:

```txt
"Create skills for React 19, TypeScript, MUI v7, Express, and Prisma"
```

The skill-creator agent will:

1. Analyze your specified tech stack (or examine your project)
2. Generate skill files in `.ai/skills/[category]/[framework]/SKILL.md`
3. Update `.github/copilot-instructions.md` placeholders with **references** to skills (not full content)
4. Provide syntax examples and best practices in separate SKILL.md files

## Skill Structure

Skills are organized by category:

```txt
skills/
├── backend/
│   └── fiber/
│       └── SKILL.md (Fiber v2 + Layered Architecture + DI)
├── database/
│   ├── postgresql/
│   │   └── SKILL.md (pgx/v5 + Connection Pooling + Batch Ops)
│   ├── redis/
│   │   └── SKILL.md (go-redis/v9 + Cache-Aside + Compression)
│   ├── sqlite/
│   │   └── SKILL.md (Read-Only SDE + Dual-DB Architecture)
│   └── migrations/
│       └── SKILL.md (golang-migrate + Testcontainers + Rollback)
├── frontend/
│   ├── nextjs/
│   │   └── SKILL.md (Next.js 16 App Router + Server Components)
│   └── radix-ui/
│       └── SKILL.md (Radix Primitives + shadcn/ui + Tailwind)
├── testing/
│   ├── go-testing/
│   │   └── SKILL.md (Table-Driven + Testcontainers + Coverage)
│   └── playwright/
│       └── SKILL.md (E2E Testing + Page Object Model)
├── tools/
│   ├── chrome-devtools/
│   │   └── SKILL.md (Browser Automation + Debugging)
│   ├── docker/
│   │   └── SKILL.md (Docker Compose + Multi-Service Orchestration)
│   └── github-mcp/
│       └── SKILL.md (GitHub API + Issue/PR Management)
└── generic/
    ├── git-workflow/
    │   └── SKILL.md (to be created as needed)
    ├── documentation-standards/
    │   └── SKILL.md (to be created as needed)
    └── agent-orchestration/
        └── SKILL.md (to be created as needed)
```

## Created Skills for This Project

### Backend: Fiber ✅

**File:** `.ai/skills/backend/fiber/SKILL.md`

- **Architecture:** Layered (Handler → Service → Repository → Database)
- **Patterns:** Dependency injection, Context propagation, Structured errors
- **Best Practices:** Handler responsibility, Timeout management, Error handling
- **Load:** `@workspace .ai/skills/backend/fiber/SKILL.md`

### Database: PostgreSQL ✅

**File:** `.ai/skills/database/postgresql/SKILL.md`

- **Tech:** pgx/v5 driver with connection pooling
- **Patterns:** Repository pattern, Transaction management, Batch operations
- **Best Practices:** Context timeouts, Prepared statements, NULL handling
- **Load:** `@workspace .ai/skills/database/postgresql/SKILL.md`

### Database: Redis ✅

**File:** `.ai/skills/database/redis/SKILL.md`

- **Tech:** go-redis/v9
- **Patterns:** Cache-aside, Dual-purpose caching (responses + computed results)
- **Best Practices:** TTL strategy, Compression (gzip), Error handling (fail-open)
- **Load:** `@workspace .ai/skills/database/redis/SKILL.md`

### Database: SQLite ✅

**File:** `.ai/skills/database/sqlite/SKILL.md`

- **Tech:** mattn/go-sqlite3 (read-only mode)
- **Patterns:** Dual-database architecture (Static SDE data)
- **Best Practices:** Read-only URI, Shared connection, Index optimization
- **Load:** `@workspace .ai/skills/database/sqlite/SKILL.md`

### Frontend: Next.js ✅

**File:** `.ai/skills/frontend/nextjs/SKILL.md`

- **Tech:** Next.js 16 App Router + React 19
- **Patterns:** Server Components (default), Client Components (`"use client"`), React Context
- **Best Practices:** Minimize client bundles, Environment variables, Loading states
- **Load:** `@workspace .ai/skills/frontend/nextjs/SKILL.md`

### Frontend: Radix UI ✅

**File:** `.ai/skills/frontend/radix-ui/SKILL.md`

- **Tech:** Radix UI + shadcn/ui + Tailwind CSS 4
- **Patterns:** Unstyled primitives, Controlled components, Composition
- **Best Practices:** Accessibility first, Variant system (CVA), Portal usage
- **Load:** `@workspace .ai/skills/frontend/radix-ui/SKILL.md`

### Testing: Go Testing ✅

**File:** `.ai/skills/testing/go-testing/SKILL.md`

- **Tech:** Go testing package + Testcontainers (PostgreSQL, Redis)
- **Patterns:** Table-driven tests, Integration tests with build tags, Migration testing
- **Best Practices:** Test pyramid (70% unit / 20% integration), Coverage-driven priorities, Real test data
- **Coverage Goals:** 13.6% → 50% (Q1) → 70% (Q2)
- **Load:** `@workspace .ai/skills/testing/go-testing/SKILL.md`

### Testing: Playwright ✅

**File:** `.ai/skills/testing/playwright/SKILL.md`

- **Tech:** Playwright v1.56.1
- **Patterns:** Page Object Model, Fixtures, API mocking
- **Best Practices:** Accessibility selectors, Auto-waiting, Test isolation
- **Load:** `@workspace .ai/skills/testing/playwright/SKILL.md`

### Tools: Chrome DevTools ✅

**File:** `.ai/skills/tools/chrome-devtools/SKILL.md`

- **Tech:** Chrome DevTools MCP Server
- **Patterns:** Snapshot-first, Element interaction, Performance tracing
- **Best Practices:** Text snapshots over screenshots, Accessibility selectors, Network monitoring
- **Load:** `@workspace .ai/skills/tools/chrome-devtools/SKILL.md`

### Tools: Docker ✅

**File:** `.ai/skills/tools/docker/SKILL.md`

- **Tech:** Docker 24+ & Docker Compose v2
- **Patterns:** Multi-service orchestration (PostgreSQL + Redis + Backend + Frontend)
- **Best Practices:** Health checks, Volume management, Rebuild before tests (CRITICAL!)
- **Load:** `@workspace .ai/skills/tools/docker/SKILL.md`

### Tools: GitHub MCP ✅

**File:** `.ai/skills/tools/github-mcp/SKILL.md`

- **Tech:** GitHub MCP Server (Official)
- **Patterns:** MCP tools over gh CLI, Search before create, Atomic PR workflow
- **Best Practices:** Batch file operations, Structured issue bodies, Review workflows
- **Load:** `@workspace .ai/skills/tools/github-mcp/SKILL.md`

### Database: Migrations ✅

**File:** `.ai/skills/database/migrations/SKILL.md`

- **Tech:** golang-migrate/migrate v4 + PostgreSQL
- **Patterns:** Sequential versioning, Up/Down pairs, Idempotent migrations
- **Best Practices:** Testcontainers testing, Safe rollback, Never modify applied migrations
- **Load:** `@workspace .ai/skills/database/migrations/SKILL.md`

## When to Create Skills

- Starting a new project with specific frameworks
- Switching to new versions (e.g., React 18 → 19)
- Adding new technologies to your stack
- Need syntax reference for less familiar tools

## When NOT Needed

- Agents work perfectly without skills
- You already know your stack well
- Just prototyping or experimenting

Skills are **optional reference material** - agents execute tasks with or without them.
