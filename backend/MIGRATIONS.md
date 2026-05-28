# ZipDesk Database Migrations

This guide explains how to run and manage ZipDesk database migrations.

## Overview

ZipDesk uses **golang-migrate** for PostgreSQL migrations. There are 7 migration sets covering:

1. **000001_init.up.sql** - Core schema (users, workspaces, subscriptions)
2. **000002_links.up.sql** - Link management (links, folders, bio pages)
3. **000003_forms.up.sql** - Form builder (forms, fields, responses, views)
4. **000004_docs.up.sql** - Document management (documents, templates, versions)
5. **000005_mail.up.sql** - Email/CRM lists (contacts, campaigns, automations)
6. **000006_crm.up.sql** - CRM pipeline (pipelines, deals, contacts, activities)
7. **000007_flow.up.sql** - Workflows & AI (events, blueprints, conversations)

## Prerequisites

### 1. Install golang-migrate

```bash
# Windows
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# macOS / Linux
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Verify installation:
```bash
migrate -version
```

### 2. Have a PostgreSQL Database

**Local Development:**
```bash
# Using Docker
docker run --name zipdesk-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=zipdesk \
  -p 5432:5432 \
  -d postgres:15
```

**Supabase (Production):**
- Create a project at https://supabase.com
- Get your database URL from Settings → Database → Connection String
- Format: `postgresql://user:password@host:5432/postgres?sslmode=require`

### 3. Update DATABASE_URL in .env

```bash
# Local
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/zipdesk?sslmode=disable

# Supabase
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@YOUR_HOST.supabase.co:5432/postgres?sslmode=require
```

## Running Migrations

### Option 1: Using Make

```bash
cd backend

# Run all pending migrations
make migrate-up

# Rollback one migration
make migrate-down

# Reset database (runs down then up)
make migrate-reset

# Check migration status
make migrate-status
```

### Option 2: Using migrate CLI Directly

```bash
cd backend

# Apply all pending migrations
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/zipdesk?sslmode=disable" \
  up

# Check current version
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/zipdesk?sslmode=disable" \
  version

# Rollback to previous version
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/zipdesk?sslmode=disable" \
  down

# Rollback all migrations
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/zipdesk?sslmode=disable" \
  down -all

# Rollback to specific version
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/zipdesk?sslmode=disable" \
  goto 5
```

### Option 3: Using Go Code

```go
import "github.com/golang-migrate/migrate/v4"

m, err := migrate.New(
    "file://migrations",
    "postgresql://postgres:postgres@localhost:5432/zipdesk?sslmode=disable",
)
if err != nil {
    log.Fatal(err)
}
defer m.Close()

if err := m.Up(); err != nil && err != migrate.ErrNoChange {
    log.Fatal(err)
}
```

## Verification

### In psql

```bash
# Connect to database
psql postgresql://postgres:postgres@localhost:5432/zipdesk

# Check schema
\dt                    # List tables
\di                    # List indexes
\d users               # Show users table structure

# Verify migrations ran
SELECT * FROM schema_migrations;
```

### In Supabase Dashboard

1. Go to your project
2. Navigate to **SQL Editor**
3. Write query:
   ```sql
   SELECT version, dirty FROM schema_migrations ORDER BY version DESC;
   ```
4. Verify all 7 versions are present and `dirty = false`

## Troubleshooting

### Connection Refused

**Error:** `error: failed to connect to database`

**Solution:**
- Verify DATABASE_URL is correct
- Check if PostgreSQL service is running
- For Docker: `docker ps | grep postgres`
- For local: Ensure port 5432 is accessible

### No such host

**Error:** `dial tcp: lookup [host]: no such host`

**Solution:**
- Verify hostname is correct in DATABASE_URL
- For Supabase: Check project settings for correct hostname
- Network connectivity may be blocked

### Migration file not found

**Error:** `error: file://migrations (.*) no such file or directory`

**Solution:**
- Ensure you're in the `backend` directory
- Verify `migrations` folder exists with SQL files
- Check file naming: `000001_init.up.sql`, `000001_init.down.sql`, etc.

### Dirty database state

**Error:** `error: Dirty database version X. Fix and force version.`

**Solution:**
```bash
# Force mark migration as clean (use with caution)
migrate -path migrations \
  -database "your-db-url" \
  version

# Then manually fix the schema issue
# and retry migrations
```

### SSL requirement

**Error:** `SSL not available` or similar SSL errors

**Solution:**
- For Supabase: Use `sslmode=require` in DATABASE_URL
- For local: Use `sslmode=disable` in DATABASE_URL
- Never use `sslmode=disable` in production

## Creating New Migrations

To add a new migration:

```bash
# Create new migration files
migrate create -ext sql -dir migrations -seq create_your_table

# This creates:
# migrations/000008_create_your_table.up.sql
# migrations/000008_create_your_table.down.sql

# Edit the SQL files
# Run migrations as normal
```

## Testing Migrations

### Test Manually

```bash
# Create test database
createdb zipdesk_test

# Run migrations
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/zipdesk_test?sslmode=disable" \
  up

# Verify structure
psql -d zipdesk_test -c "\dt"

# Cleanup
dropdb zipdesk_test
```

### With Go Tests

```go
package tests

import (
    "testing"
    "github.com/golang-migrate/migrate/v4"
)

func TestMigrations(t *testing.T) {
    m, err := migrate.New(
        "file://../migrations",
        "postgresql://postgres:postgres@localhost:5432/zipdesk_test?sslmode=disable",
    )
    if err != nil {
        t.Fatal(err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        t.Fatal(err)
    }

    version, dirty, err := m.Version()
    if err != nil {
        t.Fatal(err)
    }

    if version != 7 || dirty {
        t.Fatalf("Expected version 7 clean, got %d dirty=%v", version, dirty)
    }
}
```

## Production Checklist

- [ ] Database URL uses `sslmode=require`
- [ ] Connection pooling is configured (max_open_conns in app)
- [ ] Backup database before running migrations
- [ ] Test migrations on staging environment first
- [ ] Monitor application after migration
- [ ] Keep rollback plan ready
- [ ] Document any manual schema changes
- [ ] Review migration impact on existing data

## Next Steps

1. **Local Testing:**
   - Set up local PostgreSQL with Docker
   - Run migrations against it
   - Verify all tables are created

2. **Production Deployment:**
   - Create Supabase project
   - Backup any existing data
   - Run migrations on production database
   - Verify schema in dashboard

3. **Integration:**
   - Update `.env` with production DATABASE_URL
   - Start application server
   - Test API endpoints against new schema
