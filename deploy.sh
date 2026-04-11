#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/srv/gometeo"
BRANCH="master"
COMPOSE_FILE="docker-compose.yml"

cd "$REPO_DIR"

if [[ "$(pwd)" != "$REPO_DIR" ]]; then
    echo "error: cwd is $(pwd), expected $REPO_DIR" >&2
    exit 1
fi

if [[ ! -d .git ]]; then
    echo "error: $REPO_DIR is not a git repository" >&2
    exit 1
fi

current_branch=$(git rev-parse --abbrev-ref HEAD)
if [[ "$current_branch" != "$BRANCH" ]]; then
    echo "error: on branch '$current_branch', expected '$BRANCH'" >&2
    exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "error: working tree has uncommitted changes" >&2
    git status --short >&2
    exit 1
fi

if [[ ! -f "$COMPOSE_FILE" ]]; then
    echo "error: $COMPOSE_FILE not found" >&2
    exit 1
fi

echo ">>> pulling $BRANCH"
git pull --ff-only origin "$BRANCH"

COMMIT_ID=$(git rev-parse --short HEAD)
export COMMIT_ID
echo ">>> building image at commit $COMMIT_ID"
docker compose -f "$COMPOSE_FILE" build

echo ">>> restarting stack"
docker compose -f "$COMPOSE_FILE" down
docker compose -f "$COMPOSE_FILE" up -d

echo ">>> done (commit $COMMIT_ID)"
docker compose -f "$COMPOSE_FILE" ps
