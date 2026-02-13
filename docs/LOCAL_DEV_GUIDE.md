# Local Development Quick Reference

## Your Local DevLake Setup

**Database**: PostgreSQL 13 running in Docker
- Container name: `devlake-postgres`
- Host: `localhost:5432`
- Database: `lake`
- User: `merico`
- Password: `merico`

**DevLake Server**:
- Port: `8080`
- URL: `http://localhost:8080`
- Config: `.env` file in this directory

---

## Common Commands

### Starting/Stopping

```bash
# Start DevLake server
make run

# Build everything (plugins + server)
make build

# Build just plugins
make build-plugin

# Build just server
make build-server

# Run with hot reload (development)
make dev
```

### Database Management

```bash
# Check database is running
docker ps | grep devlake-postgres

# Start database (if stopped)
docker start devlake-postgres

# Stop database
docker stop devlake-postgres

# Access database directly
docker exec -it devlake-postgres psql -U merico -d lake

# View database logs
docker logs devlake-postgres
```

### Testing

```bash
# Run all tests
make test

# Run unit tests only
make unit-test

# Run specific plugin tests
go test ./plugins/jira/...
go test ./plugins/github/...

# Run your new plugin tests
go test ./plugins/businessmetrics/...
go test ./plugins/findevops/...
```

### Code Quality

```bash
# Run linter
make lint

# Format code
make fmt

# Generate mocks (before running tests)
make mock

# Generate swagger docs
make swag
```

---

## File Structure for New Plugins

```
plugins/
├── businessmetrics/          # Business alignment plugin
│   ├── impl/impl.go         # Plugin registration
│   ├── api/                 # HTTP endpoints
│   ├── tasks/               # Data collection/processing
│   ├── models/              # Data models
│   │   └── migrationscripts/  # Database migrations
│   └── e2e/                 # End-to-end tests
│
├── findevops/               # Cost & capitalization plugin
│   ├── impl/impl.go
│   ├── api/
│   ├── tasks/
│   ├── models/
│   │   └── migrationscripts/
│   └── e2e/
│
├── aidetector/              # AI impact detection plugin
│   └── ...
│
└── capacityplanner/         # Capacity planning plugin
    └── ...
```

---

## Troubleshooting

### Build Errors

**Error**: `cannot find package`
```bash
# Solution: Download dependencies
go mod download
go mod tidy
```

**Error**: `database connection failed`
```bash
# Solution: Check database is running
docker ps | grep postgres
# If not running:
docker start devlake-postgres
```

**Error**: `port 8080 already in use`
```bash
# Solution: Check what's using the port
lsof -i :8080
# Kill the process or change PORT in .env
```

### Database Issues

**Reset database** (WARNING: Deletes all data):
```bash
docker stop devlake-postgres
docker rm devlake-postgres
# Then recreate with the docker run command from setup
```

**Check tables exist**:
```bash
docker exec -it devlake-postgres psql -U merico -d lake -c "\\dt"
```

---

## Connecting to Production Data Sources

To test with real data, update `.env`:

```bash
# Jira
JIRA_ENDPOINT=https://your-company.atlassian.net
JIRA_AUTH_TOKEN=your_token_here

# GitHub
GITHUB_ENDPOINT=https://api.github.com
GITHUB_AUTH=your_github_pat_here
```

Then configure connections via:
1. Web UI: `http://localhost:8080`
2. API: `POST /api/connections`

---

## Development Workflow

### Adding a New Plugin

1. **Create directory structure**:
   ```bash
   mkdir -p plugins/yourplugin/{impl,api,tasks,models/migrationscripts,e2e}
   ```

2. **Create impl.go** (plugin registration):
   ```go
   package impl

   import "github.com/apache/incubator-devlake/core/plugin"

   var _ plugin.PluginMeta = (*YourPlugin)(nil)

   type YourPlugin struct{}

   func (p YourPlugin) Description() string {
       return "Your plugin description"
   }
   // ... implement other interfaces
   ```

3. **Create models** in `models/`:
   - Define Go structs for database tables
   - Table names prefixed with `_tool_yourplugin_`

4. **Create migration** in `models/migrationscripts/`:
   - Named: `YYYYMMDD_description.go`
   - Implements `script.Script` interface

5. **Create tasks** in `tasks/`:
   - Collector: Fetch data from API
   - Extractor: Parse raw data
   - Convertor: Transform to domain layer

6. **Build and test**:
   ```bash
   make build-plugin
   make build-server
   make run
   ```

---

## Useful Database Queries

```sql
-- Check Jira issues
SELECT type, COUNT(*)
FROM _tool_jira_board_issues
GROUP BY type;

-- Check recent PRs
SELECT title, created_date
FROM pull_requests
ORDER BY created_date DESC
LIMIT 10;

-- Check domain layer data
SELECT * FROM issues LIMIT 10;
SELECT * FROM pull_requests LIMIT 10;

-- Check your new tables (after migration)
SELECT * FROM _tool_businessmetrics_initiatives LIMIT 10;
SELECT * FROM domain_business_initiatives LIMIT 10;
```

---

## Next Steps After Setup

1. ✅ Local environment running
2. Create plugin directory structure
3. Write domain models
4. Write database migrations
5. Implement data collection tasks
6. Write tests
7. Build and run locally
8. Test with production data

---

## Resources

- **Full Plan**: `../DEVLAKE_EXTENSION_PLAN.md`
- **Project Overview**: `../CLAUDE.md`
- **DevLake Docs**: https://devlake.apache.org/docs/DeveloperManuals/PluginImplementation
- **Makefile**: See `../Makefile` for all available commands
