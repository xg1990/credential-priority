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

	// 999 提权仅看 LongWindowResetAt（周额度临近刷新），忽略 5h 短窗（避免常态 999 提权）。
	// 当 LongWindowResetAt 存在且在 ResetBoostWithin（如 24h）内即将重置时，才触发提权。
	switch provider {
	case core.ProviderClaude, core.ProviderAntigravity, core.ProviderCodex, core.ProviderXAI:
		resetAt = item.LongWindowResetAt
	default:
		resetAt = item.LongWindowResetAt
		if resetAt == nil {
			resetAt = item.ResetAt
		}
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
