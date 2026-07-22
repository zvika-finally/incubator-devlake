#!/bin/bash
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Clear any stale unified-search (bleve) index before starting Grafana.
#
# Grafana keeps its unified-search index under $GF_PATHS_DATA (/var/lib/grafana),
# which in our deployment is a shared EFS volume. bleve takes an exclusive lock on
# the index directory; when a task is stopped ungracefully the lock is orphaned on
# the volume, and the next task fails to start with:
#   "index is locked by another process ... err=timeout"
# which fails the health check and deadlocks the ECS rollout (old task keeps the
# lock, new task can never come up). The index is rebuilt from the database on
# boot, so it is safe to remove any leftover copy first. This makes startup
# self-healing regardless of how the previous task exited.
GF_DATA_DIR="${GF_PATHS_DATA:-/var/lib/grafana}"
rm -rf "${GF_DATA_DIR}/unified-search" 2>/dev/null || true

DATASOURCE_FILE="/etc/grafana/provisioning/datasources/datasource.yml"

# Detect database type
# Priority: DATABASE_TYPE, then legacy MYSQL_*/POSTGRES_* auto-detection
if [ -n "$DATABASE_TYPE" ]; then
  MODE="$DATABASE_TYPE"

  case "$MODE" in
    mysql)
      export MYSQL_URL="${DATABASE_HOST:-mysql}:${DATABASE_PORT:-3306}"
      export MYSQL_DATABASE="${DATABASE_NAME:-lake}"
      export MYSQL_USER="${DATABASE_USER:-merico}"
      export MYSQL_PASSWORD="${DATABASE_PASSWORD:-merico}"
      ;;
    postgresql)
      export POSTGRES_URL="${DATABASE_HOST:-postgres}:${DATABASE_PORT:-5432}"
      export POSTGRES_DATABASE="${DATABASE_NAME:-lake}"
      export POSTGRES_USER="${DATABASE_USER:-merico}"
      export POSTGRES_PASSWORD="${DATABASE_PASSWORD:-merico}"
      ;;
    *)
      echo "ERROR: DATABASE_TYPE must be 'mysql' or 'postgresql'"
      exit 1
      ;;
  esac
else
  # Legacy: auto-detect from MYSQL_*/POSTGRES_* vars
  if [ -n "$POSTGRES_URL" ]; then
    MODE="postgresql"
  elif [ -n "$MYSQL_URL" ]; then
    MODE="mysql"
  else
    echo "WARNING: No database vars. Defaulting to mysql."
    MODE="mysql"
    export MYSQL_URL="mysql:3306"
    export MYSQL_DATABASE="lake"
    export MYSQL_USER="merico"
    export MYSQL_PASSWORD="merico"
  fi
fi

echo "Database type: $MODE"

# Remove unused dashboard folder to prevent confusion
if [ "$MODE" = "mysql" ]; then
  rm -rf /etc/grafana/dashboards/postgresql
else
  rm -rf /etc/grafana/dashboards/mysql
  SSL_MODE="${DATABASE_SSL_MODE:-disable}"
  echo "SSL Mode: ${SSL_MODE}"
fi

# Locate Homepage.json for the home dashboard. The layout differs by deployment:
#  - baked-in image keeps variant subfolders: /etc/grafana/dashboards/<mode>/Homepage.json
#  - dev compose bind-mounts the variant folder directly: /etc/grafana/dashboards/Homepage.json
# Probe both so the home dashboard resolves in either case (a wrong path makes
# Grafana 12+ return HTTP 500 "Failed to load home dashboard").
for _hp in \
  "/etc/grafana/dashboards/${MODE}/Homepage.json" \
  "/etc/grafana/dashboards/Homepage.json"; do
  if [ -f "$_hp" ]; then
    export GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH="$_hp"
    break
  fi
done

echo "Homepage: $GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH"

# --- Legacy persistent-volume hygiene -------------------------------------
# This image runs as an arbitrary uid with gid 0 (OpenShift-compatible). Volumes
# created by older dashboard images (uid 472, file mode 0640) can be read-only for
# the current runtime user, which makes Grafana's DB migrations fail with
# "attempt to write a readonly database". Best-effort self-heal here; if it can't
# be fixed (non-root, non-owner) print a clear, actionable error instead of a
# cryptic SQLite crash.
GRAFANA_DATA_DIR="${GF_PATHS_DATA:-/var/lib/grafana}"
GRAFANA_DB="${GRAFANA_DATA_DIR}/grafana.db"

if [ -e "$GRAFANA_DB" ] && [ ! -w "$GRAFANA_DB" ]; then
  echo "WARNING: ${GRAFANA_DB} not writable by $(id -u):$(id -g); attempting permission fix..."
  chgrp -R 0 "$GRAFANA_DATA_DIR" 2>/dev/null || true
  chmod -R g+rwX "$GRAFANA_DATA_DIR" 2>/dev/null || true
fi

if [ -e "$GRAFANA_DB" ] && [ ! -w "$GRAFANA_DB" ]; then
  echo "ERROR: ${GRAFANA_DB} is still read-only for gid 0."
  echo "       The persistent volume was likely created by an older image (uid 472, mode 0640)."
  echo "       Fix it once from the host, then recreate this container:"
  echo "         docker run --rm -v <project>_grafana-storage:/data alpine \\"
  echo "           sh -c 'chgrp -R 0 /data && chmod -R g+rwX /data'"
fi

# Remove the deprecated Angular grafana-piechart-panel plugin if it lingers in a
# legacy volume. Grafana 12+ dropped Angular support and dashboards now use the
# core "piechart" panel; leaving it causes noisy plugin-validation errors.
LEGACY_PIECHART="${GRAFANA_DATA_DIR}/plugins/grafana-piechart-panel"
if [ -d "$LEGACY_PIECHART" ]; then
  echo "Removing deprecated grafana-piechart-panel from data volume..."
  rm -rf "$LEGACY_PIECHART" 2>/dev/null || true
fi
# --------------------------------------------------------------------------

# Provision the datasource via a CONFIG FILE (not the HTTP API).
#
# Basic auth is disabled in this deployment (Okta-only: GF_AUTH_BASIC_ENABLED=false),
# so creating the datasource through admin:password against /api/datasources returns
# 401 and the datasource is never created -- leaving every dashboard and template
# variable with no datasource and therefore no data. File-based provisioning runs at
# startup and needs no API auth. `deleteDatasources` clears any stale/duplicate entry
# of the same name left in a persistent volume so our fixed UID stays authoritative.
if [ "$MODE" = "mysql" ]; then
  cat > "$DATASOURCE_FILE" <<DSYAML
apiVersion: 1
deleteDatasources:
  - name: mysql
    orgId: 1
datasources:
  - uid: devlake-mysql-api
    name: mysql
    type: mysql
    access: proxy
    url: "${MYSQL_URL}"
    database: "${MYSQL_DATABASE}"
    user: "${MYSQL_USER}"
    isDefault: true
    editable: true
    secureJsonData:
      password: "${MYSQL_PASSWORD}"
DSYAML
else
  SSL_MODE="${DATABASE_SSL_MODE:-disable}"
  cat > "$DATASOURCE_FILE" <<DSYAML
apiVersion: 1
deleteDatasources:
  - name: postgresql
    orgId: 1
datasources:
  - uid: devlake-postgres-api
    name: postgresql
    type: grafana-postgresql-datasource
    access: proxy
    url: "${POSTGRES_URL}"
    database: "${POSTGRES_DATABASE}"
    user: "${POSTGRES_USER}"
    isDefault: true
    editable: true
    jsonData:
      sslmode: "${SSL_MODE}"
      postgresVersion: 1400
      database: "${POSTGRES_DATABASE}"
    secureJsonData:
      password: "${POSTGRES_PASSWORD}"
DSYAML
fi

echo "Datasource provisioned to ${DATASOURCE_FILE} (uid devlake-${MODE}-api)"

# Start Grafana in the foreground. The datasource is file-provisioned above (no API
# auth required). exec gives Grafana PID 1 so it receives SIGTERM and shuts down
# cleanly -- which also releases the unified-search index lock on the shared volume.
exec /run.sh "$@"
