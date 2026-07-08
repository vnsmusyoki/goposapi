#!/usr/bin/env bash
#
# Migration helper script for the POS API.
# Wraps the golang-migrate CLI so you don't need to remember flags.
#
# Usage:
#   ./scripts/migrate.sh up               # apply all pending migrations
#   ./scripts/migrate.sh down             # roll back the last migration
#   ./scripts/migrate.sh down-all         # roll back everything
#   ./scripts/migrate.sh create <name>    # create a new migration pair
#   ./scripts/migrate.sh version          # show current migration version
#   ./scripts/migrate.sh force <version>  # force db to a specific version (fix dirty state)
#   ./scripts/migrate.sh drop             # drop everything in the db (dangerous)
#
# Requires: golang-migrate CLI installed
#   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
#
# Requires DATABASE_URL to be set, e.g. in a .env file:
#   DATABASE_URL=postgres://user:password@localhost:5432/pos_db?sslmode=disable

set -euo pipefail

# Load env files if present.
# Use shell sourcing so DATABASE_URL can reference DB_* variables.
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

if [ -f .env.local ]; then
  set -a
  . ./.env.local
  set +a
fi

if [ -z "${DATABASE_URL:-}" ]; then
  echo "Error: DATABASE_URL is not set. Add it to your .env file or export it."
  exit 1
fi

MIGRATIONS_DIR="./migrations"
CMD="${1:-}"

if ! command -v migrate &> /dev/null; then
  echo "Error: 'migrate' CLI not found."
  echo "Install it with:"
  echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
  exit 1
fi

case "$CMD" in
  up)
    echo "Applying all pending migrations..."
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
    ;;

  down)
    echo "Rolling back last migration..."
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down 1
    ;;

  down-all)
    echo "Rolling back ALL migrations..."
    read -p "Are you sure? This will drop all migrated tables. [y/N] " confirm
    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
      migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down -all
    else
      echo "Cancelled."
    fi
    ;;

  create)
    NAME="${2:-}"
    if [ -z "$NAME" ]; then
      echo "Usage: ./scripts/migrate.sh create <migration_name>"
      exit 1
    fi
    migrate create -ext sql -dir "$MIGRATIONS_DIR" "$NAME"
    echo "Created migration files for: $NAME"
    ;;

  version)
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version
    ;;

  force)
    VERSION="${2:-}"
    if [ -z "$VERSION" ]; then
      echo "Usage: ./scripts/migrate.sh force <version>"
      exit 1
    fi
    migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" force "$VERSION"
    ;;

  drop)
    echo "This will DROP ALL TABLES in the database."
    read -p "Type 'yes' to confirm: " confirm
    if [ "$confirm" == "yes" ]; then
      migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" drop -f
    else
      echo "Cancelled."
    fi
    ;;

  *)
    echo "Usage: ./scripts/migrate.sh {up|down|down-all|create <name>|version|force <version>|drop}"
    exit 1
    ;;
esac
