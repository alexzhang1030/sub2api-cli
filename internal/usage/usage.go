package usage

import (
	"sort"
	"strings"
	"time"

	"github.com/alex/sub2api-cli/internal/sub2api"
)

type Input struct {
	Profile     string
	Date        string
	Timezone    string
	Generated   time.Time
	ModelFilter string
	Stats       sub2api.DashboardStats
	Trend       []sub2api.TrendDataPoint
	Models      []sub2api.ModelStat
}

type Report struct {
	Profile     string
	Date        string
	Timezone    string
	Generated   time.Time
	ModelFilter string
	Totals      Totals
	Trend       []sub2api.TrendDataPoint
	Models      []sub2api.ModelStat
}

type Totals struct {
	Requests       int64
	TotalTokens    int64
	ActualCost     float64
	StandardCost   float64
	TokenBreakdown TokenBreakdown
	RPM            int64
	TPM            int64
	AverageLatency float64
}

type TokenBreakdown struct {
	Input         int64
	Output        int64
	CacheCreation int64
	CacheRead     int64
}

func BuildTodayReport(in Input) Report {
	return buildReport(in, totalsFromTodayStats(in.Stats))
}

func BuildAllReport(in Input) Report {
	return buildReport(in, totalsFromTotalStats(in.Stats))
}

func buildReport(in Input, baseTotals Totals) Report {
	models := filterModels(in.Models, in.ModelFilter)
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].ActualCost == models[j].ActualCost {
			return models[i].TotalTokens > models[j].TotalTokens
		}
		return models[i].ActualCost > models[j].ActualCost
	})
	totals := baseTotals
	if strings.TrimSpace(in.ModelFilter) != "" {
		totals = totalsFromModels(models)
	}
	return Report{
		Profile:     in.Profile,
		Date:        in.Date,
		Timezone:    in.Timezone,
		Generated:   in.Generated,
		ModelFilter: strings.TrimSpace(in.ModelFilter),
		Totals:      totals,
		Trend:       append([]sub2api.TrendDataPoint(nil), in.Trend...),
		Models:      models,
	}
}

func WithModelFilter(report Report, filter string) Report {
	filter = strings.TrimSpace(filter)
	out := report
	out.ModelFilter = filter
	out.Models = filterModels(report.Models, filter)
	sort.SliceStable(out.Models, func(i, j int) bool {
		if out.Models[i].ActualCost == out.Models[j].ActualCost {
			return out.Models[i].TotalTokens > out.Models[j].TotalTokens
		}
		return out.Models[i].ActualCost > out.Models[j].ActualCost
	})
	if filter != "" {
		out.Totals = totalsFromModels(out.Models)
	}
	return out
}

func filterModels(models []sub2api.ModelStat, filter string) []sub2api.ModelStat {
	filter = strings.ToLower(strings.TrimSpace(filter))
	out := make([]sub2api.ModelStat, 0, len(models))
	for _, model := range models {
		if filter == "" || filter == "all" || strings.Contains(strings.ToLower(model.Model), filter) {
			out = append(out, model)
		}
	}
	return out
}

func totalsFromTodayStats(stats sub2api.DashboardStats) Totals {
	return Totals{
		Requests:       stats.TodayRequests,
		TotalTokens:    stats.TodayTokens,
		ActualCost:     stats.TodayActualCost,
		StandardCost:   stats.TodayCost,
		RPM:            stats.RPM,
		TPM:            stats.TPM,
		AverageLatency: stats.AverageDurationMs,
		TokenBreakdown: TokenBreakdown{
			Input:         stats.TodayInputTokens,
			Output:        stats.TodayOutputTokens,
			CacheCreation: stats.TodayCacheCreationTokens,
			CacheRead:     stats.TodayCacheReadTokens,
		},
	}
}

func totalsFromTotalStats(stats sub2api.DashboardStats) Totals {
	return Totals{
		Requests:       stats.TotalRequests,
		TotalTokens:    stats.TotalTokens,
		ActualCost:     stats.TotalActualCost,
		StandardCost:   stats.TotalCost,
		RPM:            stats.RPM,
		TPM:            stats.TPM,
		AverageLatency: stats.AverageDurationMs,
		TokenBreakdown: TokenBreakdown{
			Input:         stats.TotalInputTokens,
			Output:        stats.TotalOutputTokens,
			CacheCreation: stats.TotalCacheCreationTokens,
			CacheRead:     stats.TotalCacheReadTokens,
		},
	}
}

func totalsFromModels(models []sub2api.ModelStat) Totals {
	var totals Totals
	for _, model := range models {
		totals.Requests += model.Requests
		totals.TotalTokens += model.TotalTokens
		totals.ActualCost += model.ActualCost
		totals.StandardCost += model.Cost
		totals.TokenBreakdown.Input += model.InputTokens
		totals.TokenBreakdown.Output += model.OutputTokens
		totals.TokenBreakdown.CacheCreation += model.CacheCreationTokens
		totals.TokenBreakdown.CacheRead += model.CacheReadTokens
	}
	return totals
}
