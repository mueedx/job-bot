#!/bin/sh
set -e

mkdir -p /data /app/resumes

# First run: seed the editable company list without overwriting user edits.
if [ ! -f /data/target_companies.yaml ]; then
  cp /app/seed/target_companies.yaml.example /data/target_companies.yaml
fi

# Resume paths are stored as resumes/<file>.pdf and resolved through RESUME_DIR.
cd /app
exec ./server
