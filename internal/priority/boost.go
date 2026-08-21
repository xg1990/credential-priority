package priority

import (
	"credential-priority/internal/core"
	"time"
)

func plannedPriority(item PlanItem, basePriority int, options Options) int {
	if resetBoost(item, options) > 0 {
		return 999
	}
	return basePriority
}

func resetBoost(item PlanItem, options Options) int {
	if options.ResetBoostWithin <= 0 || options.ResetBoost <= 0 {
		return 0
	}

	provider := planItemProvider(item)
	var resetAt *time.Time

	// Claude / Antigravity / Codex：短窗 ResetAt 临近刷新优先，否则退回 LongWindowResetAt，
	// 两者都不满足时退回 ResetAt（可能为 nil）——三者都有对称的 5h 短窗 + weekly 长窗结构。
	// xAI：999 提权仅看 LongWindowResetAt（free 计划另有独立冷却机制，短窗不参与提权判断）。
	switch provider {
	case core.ProviderClaude, core.ProviderAntigravity, core.ProviderCodex:
		if item.ResetAt != nil && item.ResetAt.After(options.Now) && item.ResetAt.Sub(options.Now) < options.ResetBoostWithin {
			resetAt = item.ResetAt
		} else if item.LongWindowResetAt != nil {
			resetAt = item.LongWindowResetAt
		} else {
			resetAt = item.ResetAt
		}
	case core.ProviderXAI:
		resetAt = item.LongWindowResetAt
	default:
		resetAt = item.ResetAt
	}

	if resetAt == nil {
		return 0
	}
	// paid：各提供商 effective ResetAt near-reset 均可提权。
	// Free/Unknown：仅 Antigravity、Codex、Claude；禁止 xAI Free（及 xAI free 计划）。
	if paidRank(item.PlanType) == 0 {
		provider := planItemProvider(item)
		if provider == core.ProviderXAI || isXAIFreePlanItem(item) {
			return 0
		}
		if provider != core.ProviderAntigravity && provider != core.ProviderCodex && provider != core.ProviderClaude {
			return 0
		}
	}
	if resetAt.After(options.Now) && resetAt.Sub(options.Now) < options.ResetBoostWithin {
		return options.ResetBoost
	}
	return 0
}
