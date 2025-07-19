#!/usr/bin/env just --justfile


# Add pre-commit to the git hooks so it auto-runs on each commit
setup-pre-commit:
    pre-commit install

# Run all pre-commit rules against all files
lint:
    pre-commit run --all-files
