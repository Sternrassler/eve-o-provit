# Docker & Docker Compose Skill

**Tech Stack:** Docker 24+ & Docker Compose v2  
**Purpose:** Multi-service container orchestration for development and testing  
**Project Services:** PostgreSQL, Redis, Backend API, Frontend (Next.js)

---

## Architecture Overview

**Docker Compose Setup** in this project provides complete development environment:

- **PostgreSQL Database:** Main application data store (port 5432)
- **Redis Cache:** Cache layer + session storage (port 6379)
- **Backend API:** Go/Fiber application (port 9001)
- **Frontend:** Next.js application (port 9000)

**When to Use:**

- Local development with all services
- Integration testing with real database/cache
- Debugging multi-service interactions
- Reproducing production-like environment

**Critical Workflow Rule:**
 **Bei JEDER Code-Änderung:** `make docker-rebuild` VOR Tests ausführen!

---

## Architecture Patterns

### 1. Development Environment as Code

**Pattern:** All services defined in `deployments/docker-compose.yml` with consistent configuration.

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: eveprovit
      POSTGRES_PASSWORD: dev
      POSTGRES_DB: eveprovit
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./init-db:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U eveprovit"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  backend:
    build:
      context: ../backend
      dockerfile: Dockerfile
    ports:
      - "9001:9001"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      DATABASE_URL: postgresql://eveprovit:dev@postgres:5432/eveprovit
      REDIS_URL: redis://redis:6379
```

**Benefits:**

- One-command environment setup
- Consistent across team members
- Production-like configuration
- Isolated from host system

### 2. Service Dependency Chain

**Pattern:** Services start in correct order using `depends_on` with health checks.

```txt
PostgreSQL → (healthy) → Redis → (healthy) → Backend → (healthy) → Frontend
```

**Anti-Pattern:** Starting backend before database is ready (connection failures).

### 3. Volume Management Strategy

**Pattern:** Named volumes for persistent data, bind mounts for development code.

```yaml
volumes:
  postgres-data:    # Persists between restarts
  redis-data:       # Persists cache data
  
services:
  backend:
    volumes:
      - ../backend:/app  # Live code reload (bind mount)
```

**Benefits:**

- Database survives container restarts
- Code changes reflected immediately
- Clean separation: data vs code

---

## Best Practices

1. **Always Rebuild After Code Changes**
   - **CRITICAL:** `make docker-rebuild` before running tests
   - Ensures latest code is in container
   - Prevents "works locally, fails in Docker" issues
   - Alternative: `make docker-down && make docker-build && make docker-up`

2. **Use Health Checks**
   - Define health checks for all services
   - Use `condition: service_healthy` in depends_on
   - Prevents premature service startup
   - Example: Backend waits for DB + Redis to be ready

3. **Named Volumes for Data**
   - Use named volumes for databases (postgres-data, redis-data)
   - Survives `docker-compose down`
   - Destroyed only with `docker-compose down -v`
   - Backup volumes before destructive operations

4. **Environment Variable Management**
   - Store in `.env` file (gitignored)
   - Reference in docker-compose.yml
   - Never commit credentials
   - Use different values per environment (dev, staging, prod)

5. **Network Isolation**
   - Default bridge network for all services
   - Services communicate via service names (not localhost)
   - Example: Backend connects to `postgres:5432`, not `localhost:5432`

6. **Log Management**
   - Use `docker-compose logs -f` for live logs
   - Filter by service: `docker-compose logs -f backend`
   - Set log rotation limits to prevent disk fill
   - Use structured logging in applications

7. **Resource Limits**
   - Set memory/CPU limits for services
   - Prevents one service consuming all resources
   - Mimics production constraints
   - Example: `mem_limit: 512m` for Redis

---

## Common Patterns

### Pattern 1: Complete Environment Start

**Scenario:** Start fresh development environment.

```bash
# Complete workflow
make docker-up

# What happens:
# 1. Pulls images if not present
# 2. Builds backend/frontend images
# 3. Creates network
# 4. Starts PostgreSQL → waits for healthy
# 5. Starts Redis → waits for healthy
# 6. Starts Backend → depends on DB + Redis
# 7. Starts Frontend

# Verify services:
make docker-ps

# Check logs:
make docker-logs           # All services
make docker-logs SERVICE=backend  # Specific service
```

**Access points:**

- Frontend: <http://localhost:9000>
- Backend API: <http://localhost:9001>
- PostgreSQL: localhost:5432 (user: eveprovit, db: eveprovit)
- Redis: localhost:6379

### Pattern 2: Code Change → Rebuild → Test Workflow

**Scenario:** Made changes to backend code, need to test.

```bash
# CRITICAL WORKFLOW (ALWAYS FOLLOW THIS ORDER):

# 1. Make code changes in backend/
vim backend/internal/services/cache.go

# 2. Rebuild Docker containers (REQUIRED!)
make docker-rebuild

# What happens:
# - Stops all services
# - Rebuilds images with new code
# - Starts services with fresh containers

# 3. Run tests
make test-be

# 4. Check logs if issues
make docker-logs SERVICE=backend
```

**Why rebuild is critical:**

- Docker images are **immutable**
- Code changes don't auto-propagate to containers
- Old code remains in running container until rebuild
- Tests against old code = false results

### Pattern 3: Database Shell Access

**Scenario:** Need to inspect database or run manual queries.

```bash
# Access PostgreSQL shell
make docker-shell-db

# Inside psql:
\dt                          # List tables
SELECT * FROM users LIMIT 5;
\q                           # Exit

# Alternative: Direct SQL file execution
docker exec -i eve-o-provit-postgres psql -U eveprovit -d eveprovit < schema.sql
```

### Pattern 4: Redis Cache Inspection

**Scenario:** Debug caching issues, inspect Redis keys.

```bash
# Access Redis CLI
make docker-shell-redis

# Inside redis-cli:
KEYS *                   # List all keys
GET cache:market:34      # Get specific key
FLUSHALL                 # Clear entire cache (dev only!)
INFO                     # Redis stats
exit
```

### Pattern 5: Clean Slate Restart

**Scenario:** Something broke, need fresh start with clean data.

```bash
# Nuclear option: Delete everything
make docker-clean

# What's deleted:
# - All containers
# - All volumes (postgres-data, redis-data)
# - All images
# - Network

# Then rebuild from scratch:
make docker-up

# Restore database from backup (if needed):
docker exec -i eve-o-provit-postgres psql -U eveprovit -d eveprovit < backup.sql
```

### Pattern 6: Service-Specific Restart

**Scenario:** Only backend changed, don't want to restart DB.

```bash
# Restart single service
docker-compose -f deployments/docker-compose.yml restart backend

# Or rebuild single service
docker-compose -f deployments/docker-compose.yml up -d --build backend
```

### Pattern 7: Migration Workflow

**Scenario:** Created new database migration, need to apply.

```bash
# 1. Ensure services are running
make docker-up

# 2. Run migrations
make migrate-up

# What happens:
# - golang-migrate connects to PostgreSQL
# - Applies pending migrations in order
# - Updates schema_migrations table

# 3. Verify migration
make docker-shell-db
\dt                    # Check new tables
SELECT * FROM schema_migrations;
```

---

## Anti-Patterns

### Skipping Rebuild After Code Changes

**Why:** Tests run against old code in container, not your changes.  
**Instead:** **ALWAYS `make docker-rebuild` before testing!**

### Using localhost in Service URLs

**Why:** Services run in Docker network, not on host.  
**Instead:** Use service names: `postgres:5432`, `redis:6379`

### Committing .env Files

**Why:** Contains credentials, database URLs, API keys.  
**Instead:** Use `.env.example` template, gitignore `.env`

### Running `docker-compose down -v` Without Backup

**Why:** Deletes ALL data (postgres-data, redis-data volumes).  
**Instead:** Backup database first: `pg_dump > backup.sql`

### Not Using Health Checks

**Why:** Backend starts before database ready → connection failures.  
**Instead:** Define health checks, use `depends_on: condition: service_healthy`

### Exposing All Ports in Production

**Why:** Security risk, unnecessary attack surface.  
**Instead:** Only expose frontend (reverse proxy to backend)

---

## Integration with Development Workflow

### With Backend Development

**Workflow:**

```
Code Change → make docker-rebuild → make test-be → make docker-logs
```

**Critical:** Rebuild ensures tests run against latest code.

### With Database Migrations

**Workflow:**

```
Create Migration → make migrate-create NAME=xxx → Edit SQL → make migrate-up → Verify
```

**Integration:** Migrations run against Docker PostgreSQL instance.

### With Frontend Development

**Workflow:**

```
Code Change → make docker-rebuild → Access http://localhost:9000 → Check browser console
```

**Live Reload:** Next.js dev server auto-reloads on file changes (if volume mounted).

### With Integration Tests

**Workflow:**

```
make docker-up → make test-be-int → make docker-logs
```

**Test Dependencies:** Integration tests connect to Docker services (PostgreSQL, Redis).

---

## Performance Considerations

1. **Build Cache Optimization**
   - Multi-stage Dockerfiles for smaller images
   - Layer caching: COPY go.mod first, then source code
   - `.dockerignore` excludes unnecessary files

2. **Volume Performance**
   - Named volumes faster than bind mounts
   - Use bind mounts only for development (live reload)
   - Production: Copy files into image, no volumes

3. **Network Performance**
   - Services on same network communicate directly
   - No overhead of port forwarding
   - Use service names, not IP addresses (DNS resolution)

4. **Memory Management**
   - Set mem_limit for each service
   - PostgreSQL: Tune shared_buffers, work_mem
   - Redis: Set maxmemory + eviction policy

---

## Security Guidelines

1. **Credential Management**
   - Never hardcode passwords in docker-compose.yml
   - Use environment variables from .env file
   - Rotate credentials regularly
   - Use different credentials per environment

2. **Network Security**
   - Don't expose PostgreSQL/Redis to host in production
   - Use reverse proxy (Nginx) for external access
   - Restrict container-to-container communication

3. **Image Security**
   - Use official images (postgres:16, redis:7-alpine)
   - Pin specific versions (not :latest)
   - Scan images for vulnerabilities: `trivy image postgres:16`
   - Update base images regularly

4. **Volume Security**
   - Encrypt volumes in production
   - Set proper file permissions
   - Regular backups to secure location

---

## Quick Reference

| Operation | Make Target | Use Case |
|-----------|-------------|----------|
| Start all services | `make docker-up` | Fresh environment |
| Stop all services | `make docker-down` | End session |
| **Rebuild after code change** | **`make docker-rebuild`** | **CRITICAL: Before tests!** |
| View logs | `make docker-logs` | Debug issues |
| Check service status | `make docker-ps` | Verify running |
| PostgreSQL shell | `make docker-shell-db` | Manual SQL queries |
| Redis shell | `make docker-shell-redis` | Inspect cache |
| Backend shell | `make docker-shell-api` | Debug container |
| Clean slate | `make docker-clean` | Nuclear restart |
| Build images only | `make docker-build` | Prepare images |
| Restart services | `make docker-restart` | Quick refresh |

---

## Common Debugging Scenarios

### Scenario: Backend Not Starting

**Steps:**

1. Check logs: `make docker-logs SERVICE=backend`
2. Verify database health: `make docker-ps` (should show "healthy")
3. Check environment variables: `docker exec eve-o-provit-api env`
4. Test database connection manually: `make docker-shell-db`

### Scenario: Database Connection Refused

**Steps:**

1. Verify PostgreSQL is running: `docker ps | grep postgres`
2. Check health status: `docker inspect eve-o-provit-postgres | grep Health`
3. Ensure backend uses correct URL: `postgres:5432` (not `localhost:5432`)
4. Check PostgreSQL logs: `docker logs eve-o-provit-postgres`

### Scenario: Redis Cache Not Working

**Steps:**

1. Verify Redis is running: `docker ps | grep redis`
2. Test connection: `make docker-shell-redis` → `PING` (should return PONG)
3. Check backend Redis URL: `redis://redis:6379` (not localhost)
4. Inspect keys: `redis-cli KEYS *`

### Scenario: Port Already in Use

**Steps:**

1. Find process using port: `lsof -i :5432` (or 6379, 9001)
2. Stop conflicting service: `sudo systemctl stop postgresql`
3. Or change port in docker-compose.yml: `"5433:5432"`

### Scenario: Code Changes Not Reflected

**Steps:**

1. **Run `make docker-rebuild`** (CRITICAL!)
2. Verify image rebuild: `docker images | grep eve-o-provit`
3. Check container age: `docker ps` (Created column should be recent)
4. If still not working: `make docker-clean && make docker-up`

---

## Makefile Integration

All Docker operations available via Makefile:

```makefile
# From project Makefile:
docker-up:       # Start all services
docker-down:     # Stop all services
docker-logs:     # View logs (SERVICE=name for specific)
docker-ps:       # Check status
docker-build:    # Build images
docker-clean:    # Nuclear cleanup
docker-restart:  # Quick restart (no rebuild)
docker-rebuild:  # CRITICAL: Rebuild + restart (after code changes)
docker-shell-api:   # Backend container shell
docker-shell-db:    # PostgreSQL shell
docker-shell-redis: # Redis CLI
```

**Usage Examples:**

```bash
make docker-up                     # Start everything
make docker-logs SERVICE=backend   # Tail backend logs
make docker-rebuild                # After code changes (CRITICAL!)
make docker-shell-db               # Access PostgreSQL
make docker-clean                  # Delete everything
```

---

**Last Updated:** 2025-11-04  
**Maintained By:** skill-creator agent  
**Critical Reminder:** **ALWAYS `make docker-rebuild` before tests after code changes!**
