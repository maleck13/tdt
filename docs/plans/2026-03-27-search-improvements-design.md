# Search Improvements Design

**Date:** 2026-03-27

## Problem

When `discover_tools` is called with broad, multi-intent queries like `"dining restaurant search table reservations, contact info messaging greeting"`, too many tools are returned. Three root causes:

1. Common terms like "search", "info", "get" match nearly every tool
2. Multi-intent queries (comma-separated) dilute scores across unrelated domains
3. No minimum score threshold filters out weak partial matches

## Changes

### 1. Stop-word filtering (`bm25.go`)

Add a set of common low-discriminative terms to filter after stemming in `tokenize()`. Terms like "search", "info", "get", "set", "list", "data", "util" appear in most tool descriptions and inflate match counts.

The stop words are filtered in their **stemmed** form so they match regardless of surface form (e.g., "searching" → "search").

### 2. Query splitting with max-score (`index.go`)

In `RankedSearch`, split `query.Text` on commas into sub-queries. Score each sub-query independently against the corpus. For each tool, take the **maximum score** across sub-queries.

- If there's only one segment (no commas), behavior is unchanged
- Empty segments after splitting are skipped
- Each sub-query is trimmed of whitespace

### 3. Default MinScore (`types.go` / `index.go`)

Add a `DefaultMinScore` constant of `0.15`. Apply it in `RankedSearch` when `opts.MinScore == 0`. Callers can override by setting `MinScore` explicitly.

## Files Modified

| File | Change |
|------|--------|
| `bm25.go` | Stop-word set + filtering in `tokenize()` |
| `index.go` | Sub-query splitting logic in `RankedSearch` |
| `types.go` | `DefaultMinScore` constant |
| `bm25_test.go` | Stop-word filtering tests |
| `index_test.go` or `ranked_search_test.go` | Comma splitting + MinScore tests |
| `golden_test.go` | Verify golden tests still pass (adjust thresholds if needed) |
