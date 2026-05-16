#!/bin/sh
# Virtualbabes Arena Entrypoint
# Ensures the persistence layer is ready before starting the Go backend.

set -e

# Create data dir if missing (Render Volume mount point)
mkdir -p "$DATA_DIR"

# Execute the command provided in the Dockerfile (server-bin)
exec "$@"