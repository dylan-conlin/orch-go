#!/bin/bash
# Quick wrapper — run from project root:
#   bash .harness/openscad/test/run-e2e.sh
cd "$(dirname "$0")/.." && bash test/test-e2e-pipeline.sh "$@"
