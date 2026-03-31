package tdt

import "sort"

// combineRRF merges two scored lists using Reciprocal Rank Fusion.
// Each list is ranked independently (highest score = rank 1).
// Combined score = 1/(k+rank_a) + 1/(k+rank_b).
// Tools appearing in only one list get only that list's contribution.
func combineRRF(listA, listB []toolScore, k int) []toolScore {
	kf := float64(k)

	// Sort each list descending by score to assign ranks.
	sortDesc := func(s []toolScore) []toolScore {
		out := make([]toolScore, len(s))
		copy(out, s)
		sort.Slice(out, func(i, j int) bool {
			return out[i].score > out[j].score
		})
		return out
	}

	rankedA := sortDesc(listA)
	rankedB := sortDesc(listB)

	type entry struct {
		serverName string
		score      float64
	}
	merged := make(map[string]*entry)

	for rank, ts := range rankedA {
		e, ok := merged[ts.toolName]
		if !ok {
			e = &entry{serverName: ts.serverName}
			merged[ts.toolName] = e
		}
		e.score += 1.0 / (kf + float64(rank+1))
	}

	for rank, ts := range rankedB {
		e, ok := merged[ts.toolName]
		if !ok {
			e = &entry{serverName: ts.serverName}
			merged[ts.toolName] = e
		}
		e.score += 1.0 / (kf + float64(rank+1))
	}

	result := make([]toolScore, 0, len(merged))
	for toolName, e := range merged {
		result = append(result, toolScore{
			toolName:   toolName,
			serverName: e.serverName,
			score:      e.score,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].score > result[j].score
	})
	return result
}

// bm25DampeningK controls the saturation point for BM25 normalization.
// With k=2, a raw max BM25 of 2 produces a top score of 0.5, and a raw max
// of 6 produces 0.75. This prevents weak single-token matches from being
// inflated to high scores when they happen to be the best match in a sub-query.
const bm25DampeningK = 2.0

// normalizeBM25 normalizes BM25 scores using dampened normalization.
// Instead of simple max-normalization (score/max → top always 1.0), it uses
// score/(max+k) which keeps scores low when the absolute match quality is poor.
// This prevents weak matches from being inflated when no tool truly matches a sub-query.
func normalizeBM25(scores []toolScore) []toolScore {
	if len(scores) == 0 {
		return scores
	}
	max := scores[0].score
	for _, s := range scores[1:] {
		if s.score > max {
			max = s.score
		}
	}
	out := make([]toolScore, len(scores))
	copy(out, scores)
	if max == 0 {
		return out
	}
	divisor := max + bm25DampeningK
	for i := range out {
		out[i].score = out[i].score / divisor
	}
	return out
}
