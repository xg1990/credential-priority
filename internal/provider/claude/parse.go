package claude

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"credential-priority/internal/core"
	"credential-priority/internal/host"
)

type claudeUsageResponse struct {
	PlanType         any                   `json:"plan_type"`
	Tier             any                   `json:"tier"`
	SubscriptionType any                   `json:"subscription_type"`
	Capabilities     []string              `json:"capabilities"`
	RateLimits       *claudeRateLimits     `json:"rate_limits"`
	RateLimit        *claudeWindow         `json:"rate_limit"`
	SessionLimit     *claudeWindow         `json:"session_limit"`
	FiveHour         *claudeWindow         `json:"five_hour"`
	Weekly           *claudeWindow         `json:"weekly"`
	Daily            *claudeWindow         `json:"daily"`
	Stats            *claudeWindow         `json:"stats"`
	ResetsAt         any                   `json:"resets_at"`
	ResetsAtCamel    any                   `json:"resetsAt"`
	ResetTime        any                   `json:"reset_time"`
	Remaining        any                   `json:"remaining"`
	RemainingQueries any                   `json:"remaining_queries"`
	Limit            any                   `json:"limit"`
	Used             any                   `json:"used"`
	UsedPercent      any                   `json:"used_percent"`
	LimitReached     any                   `json:"limit_reached"`
	UUID             string                `json:"uuid"`
	OrganizationUUID string                `json:"organization_uuid"`
	Error            *claudeErrorContainer `json:"error"`
}

type claudeRateLimits struct {
	SessionLimit *claudeWindow `json:"session_limit"`
	FiveHour     *claudeWindow `json:"five_hour"`
	FiveHourAlt  *claudeWindow `json:"5h"`
	Weekly       *claudeWindow `json:"weekly"`
	Daily        *claudeWindow `json:"daily"`
	Primary      *claudeWindow `json:"primary"`
	Secondary    *claudeWindow `json:"secondary"`
}

type claudeWindow struct {
	ResetsAt           any `json:"resets_at"`
	ResetsAtCamel      any `json:"resetsAt"`
	ResetTime          any `json:"reset_time"`
	ResetAfterSeconds  any `json:"reset_after_seconds"`
	LimitWindowSeconds any `json:"limit_window_seconds"`
	Remaining          any `json:"remaining"`
	RemainingQueries   any `json:"remaining_queries"`
	Limit              any `json:"limit"`
	Used               any `json:"used"`
	UsedPercent        any `json:"used_percent"`
	LimitReached       any `json:"limit_reached"`
	Name               any `json:"name"`
	Type               any `json:"type"`
}

type claudeErrorContainer struct {
	Type          string `json:"type"`
	Message       string `json:"message"`
	ResetsAt      any    `json:"resets_at"`
	ResetsAtCamel any    `json:"resetsAt"`
	ResetTime     any    `json:"reset_time"`
}

type effectiveWindow struct {
	resetAt           *time.Time
	remaining         int64
	windowType        WindowType
	longWindowResetAt *time.Time
}

// ParseClaudeUsage 将 Claude 响应 JSON 解析为可信额度 fresh probe 结果。
func ParseClaudeUsage(raw []byte, observedAt time.Time) ProbeResult {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return failedResult(observedAt, "empty claude usage response")
	}

	// 数组形态：可能为 /organizations 列表
	if trimmed[0] == '[' {
		var orgList []claudeUsageResponse
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&orgList); err == nil && len(orgList) > 0 {
			for _, org := range orgList {
				res, ok := parseSingleUsage(org, observedAt)
				if ok {
					if org.UUID != "" {
						res.OrganizationUUID = org.UUID
					} else if org.OrganizationUUID != "" {
						res.OrganizationUUID = org.OrganizationUUID
					}
					return res
				}
			}
			return failedResult(observedAt, "no quota info in organization list")
		}
	}

	var usage claudeUsageResponse
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&usage); err != nil {
		return failedResult(observedAt, "parse claude usage failed")
	}

	// 检查是否包含顶层或嵌套错误
	if usage.Error != nil {
		return ParseClaudeRateLimitError(trimmed, nil, observedAt)
	}

	res, ok := parseSingleUsage(usage, observedAt)
	if ok {
		return res
	}

	return failedResult(observedAt, "trusted claude quota window unavailable")
}

func parseSingleUsage(usage claudeUsageResponse, observedAt time.Time) (ProbeResult, bool) {
	planType := inferClaudePlanType(usage)
	orgUUID := usage.UUID
	if orgUUID == "" {
		orgUUID = usage.OrganizationUUID
	}

	// 1. 尝试从嵌套 rate_limits 提取
	if usage.RateLimits != nil {
		if win, ok := pickFromRateLimits(*usage.RateLimits, observedAt); ok {
			return ProbeResult{
				Provider:          core.ProviderClaude,
				ObservedAt:        observedAt.UTC(),
				ResetAt:           win.resetAt,
				Remaining:         int64Ptr(win.remaining),
				Window:            win.windowType,
				LongWindowResetAt: win.longWindowResetAt,
				Freshness:         core.FreshnessFresh,
				ProbeStatus:       core.ProbeStatusReady,
				Status:            StatusReady,
				PlanType:          planType,
				OrganizationUUID:  orgUUID,
			}, true
		}
	}

	// 2. 尝试从具体命名的窗口字段提取
	for _, candidate := range []struct {
		win  *claudeWindow
		wtyp WindowType
	}{
		{usage.FiveHour, WindowFiveHour},
		{usage.SessionLimit, WindowFiveHour},
		{usage.RateLimit, WindowFiveHour},
		{usage.Weekly, WindowWeekly},
		{usage.Daily, WindowDaily},
		{usage.Stats, WindowFiveHour},
	} {
		if candidate.win != nil && hasWindowData(*candidate.win) {
			resetAt := windowResetTime(observedAt, *candidate.win)
			remaining, ok := windowRemaining(*candidate.win)
			if resetAt != nil && ok {
				var longReset *time.Time
				if usage.Weekly != nil && hasWindowData(*usage.Weekly) {
					longReset = windowResetTime(observedAt, *usage.Weekly)
				}
				return ProbeResult{
					Provider:          core.ProviderClaude,
					ObservedAt:        observedAt.UTC(),
					ResetAt:           resetAt,
					Remaining:         int64Ptr(remaining),
					Window:            candidate.wtyp,
					LongWindowResetAt: longReset,
					Freshness:         core.FreshnessFresh,
					ProbeStatus:       core.ProbeStatusReady,
					Status:            StatusReady,
					PlanType:          planType,
					OrganizationUUID:  orgUUID,
				}, true
			}
		}
	}

	// 3. 尝试从顶层字段提取
	topWindow := claudeWindow{
		ResetsAt:         usage.ResetsAt,
		ResetsAtCamel:    usage.ResetsAtCamel,
		ResetTime:        usage.ResetTime,
		Remaining:        usage.Remaining,
		RemainingQueries: usage.RemainingQueries,
		Limit:            usage.Limit,
		Used:             usage.Used,
		UsedPercent:      usage.UsedPercent,
		LimitReached:     usage.LimitReached,
	}
	if hasWindowData(topWindow) {
		resetAt := windowResetTime(observedAt, topWindow)
		remaining, ok := windowRemaining(topWindow)
		if resetAt != nil && ok {
			return ProbeResult{
				Provider:         core.ProviderClaude,
				ObservedAt:       observedAt.UTC(),
				ResetAt:          resetAt,
				Remaining:        int64Ptr(remaining),
				Window:           WindowFiveHour,
				Freshness:        core.FreshnessFresh,
				ProbeStatus:      core.ProbeStatusReady,
				Status:           StatusReady,
				PlanType:         planType,
				OrganizationUUID: orgUUID,
			}, true
		}
	}

	return ProbeResult{}, false
}

func pickFromRateLimits(limits claudeRateLimits, observedAt time.Time) (effectiveWindow, bool) {
	var fiveHour *claudeWindow
	for _, w := range []*claudeWindow{limits.FiveHour, limits.FiveHourAlt, limits.SessionLimit, limits.Primary} {
		if w != nil && hasWindowData(*w) {
			fiveHour = w
			break
		}
	}
	var weekly *claudeWindow
	for _, w := range []*claudeWindow{limits.Weekly, limits.Secondary} {
		if w != nil && hasWindowData(*w) {
			weekly = w
			break
		}
	}

	var fiveHourWin, weeklyWin *parsedWindow
	if fiveHour != nil {
		resetAt := windowResetTime(observedAt, *fiveHour)
		remaining, ok := windowRemaining(*fiveHour)
		if resetAt != nil && ok {
			fiveHourWin = &parsedWindow{resetAt: resetAt, remaining: remaining}
		}
	}
	if weekly != nil {
		resetAt := windowResetTime(observedAt, *weekly)
		remaining, ok := windowRemaining(*weekly)
		if resetAt != nil && ok {
			weeklyWin = &parsedWindow{resetAt: resetAt, remaining: remaining}
		}
	}

	if fiveHourWin != nil && weeklyWin != nil {
		if weeklyWin.remaining <= 0 {
			return effectiveWindow{
				resetAt:    weeklyWin.resetAt,
				remaining:  0,
				windowType: WindowWeekly,
			}, true
		}
		if fiveHourWin.remaining <= 0 {
			return effectiveWindow{
				resetAt:           fiveHourWin.resetAt,
				remaining:         0,
				windowType:        WindowFiveHour,
				longWindowResetAt: weeklyWin.resetAt,
			}, true
		}
		return effectiveWindow{
			resetAt:           fiveHourWin.resetAt,
			remaining:         fiveHourWin.remaining,
			windowType:        WindowFiveHour,
			longWindowResetAt: weeklyWin.resetAt,
		}, true
	}

	if fiveHourWin != nil {
		return effectiveWindow{
			resetAt:    fiveHourWin.resetAt,
			remaining:  fiveHourWin.remaining,
			windowType: WindowFiveHour,
		}, true
	}

	if weeklyWin != nil {
		return effectiveWindow{
			resetAt:           weeklyWin.resetAt,
			remaining:         weeklyWin.remaining,
			windowType:        WindowWeekly,
			longWindowResetAt: weeklyWin.resetAt,
		}, true
	}

	return effectiveWindow{}, false
}

type parsedWindow struct {
	resetAt   *time.Time
	remaining int64
}

// ParseClaudeRateLimitError 解析 HTTP 429 或 rate limit error 正文与响应头。
func ParseClaudeRateLimitError(raw []byte, headers host.Header, observedAt time.Time) ProbeResult {
	var resetAt *time.Time
	var planType core.PlanType = core.PlanTypeUnknown
	zeroRemaining := int64(0)

	// 1. 尝试解析 JSON 错误体
	if len(raw) > 0 {
		var generic map[string]any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&generic); err == nil {
			resetAt = extractResetFromGenericMap(generic, observedAt)
			planType = inferPlanFromGenericMap(generic)
		}
	}

	// 2. 尝试从 headers 解析
	if resetAt == nil && headers != nil {
		resetAt = extractResetFromHeaders(headers, observedAt)
	}

	// 3. 若仍无 resetAt，提供默认 5 小时重置点
	if resetAt == nil {
		defaultReset := observedAt.UTC().Add(5 * time.Hour)
		resetAt = &defaultReset
	}

	return ProbeResult{
		Provider:    core.ProviderClaude,
		ObservedAt:  observedAt.UTC(),
		ResetAt:     resetAt,
		Remaining:   &zeroRemaining,
		Window:      WindowFiveHour,
		Freshness:   core.FreshnessFresh,
		ProbeStatus: core.ProbeStatusReady,
		Status:      StatusReady,
		PlanType:    planType,
		Error:       "rate limit reached",
	}
}

func extractResetFromGenericMap(m map[string]any, observedAt time.Time) *time.Time {
	// 直接键查找
	for _, key := range []string{"resets_at", "resetsAt", "reset_at", "reset_time", "reset_after_seconds"} {
		if v, ok := m[key]; ok {
			if t, okTime := parseAnyTime(v); okTime {
				return t
			}
			if secs, okSecs := toInt64(v); okSecs && secs > 0 {
				t := observedAt.UTC().Add(time.Duration(secs) * time.Second)
				return &t
			}
		}
	}

	// 嵌套 error 或 rate_limit 查找
	for _, nestedKey := range []string{"error", "rate_limit", "rate_limits", "detail", "data"} {
		if nested, ok := m[nestedKey].(map[string]any); ok {
			if t := extractResetFromGenericMap(nested, observedAt); t != nil {
				return t
			}
		}
	}

	return nil
}

func extractResetFromHeaders(headers host.Header, observedAt time.Time) *time.Time {
	for key, values := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		for _, val := range values {
			trimmed := strings.TrimSpace(val)
			if trimmed == "" {
				continue
			}
			switch lowerKey {
			case "anthropic-ratelimit-requests-reset", "anthropic-ratelimit-tokens-reset", "x-ratelimit-reset", "x-ratelimit-reset-requests":
				if t, ok := parseTimeString(trimmed); ok {
					return t
				}
			case "retry-after":
				if secs, err := strconv.ParseInt(trimmed, 10, 64); err == nil && secs > 0 {
					t := observedAt.UTC().Add(time.Duration(secs) * time.Second)
					return &t
				}
				if t, ok := parseTimeString(trimmed); ok {
					return t
				}
			}
		}
	}
	return nil
}

func inferPlanFromGenericMap(m map[string]any) core.PlanType {
	for _, key := range []string{"plan_type", "plan", "tier", "subscription"} {
		if v, ok := m[key]; ok {
			if s, okStr := toString(v); okStr {
				if pt := inferPlanType(s); pt != core.PlanTypeUnknown {
					return pt
				}
			}
		}
	}
	if caps, ok := m["capabilities"].([]any); ok {
		for _, c := range caps {
			if s, okStr := toString(c); okStr {
				if pt := inferPlanType(s); pt != core.PlanTypeUnknown {
					return pt
				}
			}
		}
	}
	return core.PlanTypeUnknown
}

func hasWindowData(window claudeWindow) bool {
	if _, ok := parseAnyTime(window.ResetsAt); ok {
		return true
	}
	if _, ok := parseAnyTime(window.ResetsAtCamel); ok {
		return true
	}
	if _, ok := parseAnyTime(window.ResetTime); ok {
		return true
	}
	if seconds, ok := toInt64(window.ResetAfterSeconds); ok && seconds > 0 {
		return true
	}
	if _, ok := toFloat64(window.Remaining); ok {
		return true
	}
	if _, ok := toFloat64(window.RemainingQueries); ok {
		return true
	}
	if _, ok := toFloat64(window.Limit); ok {
		return true
	}
	if _, ok := toFloat64(window.Used); ok {
		return true
	}
	if _, ok := toFloat64(window.UsedPercent); ok {
		return true
	}
	if _, ok := toBool(window.LimitReached); ok {
		return true
	}
	return false
}

func windowResetTime(observedAt time.Time, window claudeWindow) *time.Time {
	if resetAt, ok := parseAnyTime(window.ResetsAt); ok {
		return resetAt
	}
	if resetAt, ok := parseAnyTime(window.ResetsAtCamel); ok {
		return resetAt
	}
	if resetAt, ok := parseAnyTime(window.ResetTime); ok {
		return resetAt
	}
	seconds, ok := toInt64(window.ResetAfterSeconds)
	if ok && seconds > 0 {
		resetAt := observedAt.UTC().Add(time.Duration(seconds) * time.Second)
		return &resetAt
	}
	return nil
}

func windowRemaining(window claudeWindow) (int64, bool) {
	if remaining, ok := toFloat64(window.Remaining); ok {
		return nonNegativeCeil(remaining), true
	}
	if remainingQueries, ok := toFloat64(window.RemainingQueries); ok {
		return nonNegativeCeil(remainingQueries), true
	}
	limit, okLimit := toFloat64(window.Limit)
	used, okUsed := toFloat64(window.Used)
	if okLimit && okUsed {
		return nonNegativeCeil(limit - used), true
	}
	if reached, ok := toBool(window.LimitReached); ok && reached {
		return 0, true
	}
	if usedPercent, ok := toFloat64(window.UsedPercent); ok {
		remainingPercent := 100 - usedPercent
		return nonNegativeCeil(remainingPercent), true
	}
	return 0, false
}

func inferClaudePlanType(usage claudeUsageResponse) core.PlanType {
	for _, capStr := range usage.Capabilities {
		lower := strings.ToLower(strings.TrimSpace(capStr))
		if strings.Contains(lower, "claude_pro") || strings.Contains(lower, "pro") {
			return core.PlanTypePro
		}
		if strings.Contains(lower, "team") || strings.Contains(lower, "enterprise") {
			return core.PlanTypeTeam
		}
	}
	for _, raw := range []any{usage.PlanType, usage.Tier, usage.SubscriptionType} {
		if s, ok := toString(raw); ok {
			pt := inferPlanType(s)
			if pt != core.PlanTypeUnknown {
				return pt
			}
		}
	}
	return core.PlanTypeUnknown
}

func inferPlanType(value string) core.PlanType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "free", "claude_free", "none":
		return core.PlanTypeFree
	case "plus":
		return core.PlanTypePlus
	case "pro", "claude_pro", "claude-pro":
		return core.PlanTypePro
	case "team", "claude_team", "enterprise":
		return core.PlanTypeTeam
	default:
		return core.PlanTypeUnknown
	}
}

func failedResult(observedAt time.Time, message string) ProbeResult {
	return ProbeResult{
		Provider:    core.ProviderClaude,
		ObservedAt:  observedAt.UTC(),
		Window:      WindowUnknown,
		Freshness:   core.FreshnessUnknown,
		ProbeStatus: core.ProbeStatusUnknown,
		Status:      StatusProbeFailed,
		PlanType:    core.PlanTypeUnknown,
		Error:       message,
	}
}

func parseAnyTime(raw any) (*time.Time, bool) {
	switch value := raw.(type) {
	case nil:
		return nil, false
	case string:
		return parseTimeString(value)
	case float64:
		return parseUnix(int64(value))
	case json.Number:
		integer, err := value.Int64()
		if err != nil {
			return nil, false
		}
		return parseUnix(integer)
	default:
		return nil, false
	}
}

func parseTimeString(value string) (*time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, false
	}
	if integer, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return parseUnix(integer)
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			utc := parsed.UTC()
			return &utc, true
		}
	}
	return nil, false
}

func parseUnix(value int64) (*time.Time, bool) {
	if value <= 0 {
		return nil, false
	}
	if value > 1_000_000_000_000 {
		parsed := time.UnixMilli(value).UTC()
		return &parsed, true
	}
	parsed := time.Unix(value, 0).UTC()
	return &parsed, true
}

func toString(raw any) (string, bool) {
	switch value := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		return trimmed, trimmed != ""
	case json.Number:
		trimmed := strings.TrimSpace(value.String())
		return trimmed, trimmed != ""
	default:
		return "", false
	}
}

func toInt64(raw any) (int64, bool) {
	switch value := raw.(type) {
	case float64:
		return int64(value), true
	case json.Number:
		integer, err := value.Int64()
		return integer, err == nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, false
		}
		integer, err := strconv.ParseInt(trimmed, 10, 64)
		return integer, err == nil
	default:
		return 0, false
	}
}

func toFloat64(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case json.Number:
		floatValue, err := value.Float64()
		return floatValue, err == nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, false
		}
		floatValue, err := strconv.ParseFloat(trimmed, 64)
		return floatValue, err == nil
	default:
		return 0, false
	}
}

func toBool(raw any) (bool, bool) {
	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return parsed, err == nil
	default:
		return false, false
	}
}

func nonNegativeCeil(value float64) int64 {
	if value <= 0 {
		return 0
	}
	return int64(math.Ceil(value))
}

func int64Ptr(value int64) *int64 {
	return &value
}
