# Plan: Ingestion & Matcher

**Branch**: `002-ingestion-matcher` | **Date**: 2026-09-07

## Summary

Go scrapers for public boards + rule-based matcher; `POST /api/ingest/run` and `GET /api/ingest/status`.

## Technical Context

Go 1.25, chi, sqlx, yaml.v3, net/http clients. SQLite jobs/matches.

## Constitution Check

PASS — public boards + local persist; no fake submit; opted-in only for later LLM.

## Structure

`internal/scrapers/*`, `internal/services/matcher.go`, `internal/services/ingest.go`, API ingest handlers.
