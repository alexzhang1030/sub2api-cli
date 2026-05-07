package usage

import (
	"sort"
	"time"

	"github.com/alex/sub2api-cli/internal/sub2api"
)

type Input struct {
	Profile   string
	Date      string
	Timezone  string
	Generated time.Time
	Stats     sub2api.DashboardStats
	Trend     []sub2api.TrendDataPoint
	Models    []sub2api.ModelStat
}

type Report struct {
	Profile   string
	Date      string
	Timezone  string
	Generated time.Time
	Totals    Totals
	Trend     []sub2api.TrendDataPoint
	Models    []sub2api.ModelStat
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
	models := append([]sub2api.ModelStat(nil), in.Models...)
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].ActualCost == models[j].ActualCost {
			return models[i].TotalTokens > models[j].TotalTokens
		}
		return models[i].ActualCost > models[j].ActualCost
	})
	return Report{
		Profile:   in.Profile,
		Date:      in.Date,
		Timezone:  in.Timezone,
		Generated: in.Generated,
		Totals: Totals{
			Requests:       in.Stats.TodayRequests,
			TotalTokens:    in.Stats.TodayTokens,
			ActualCost:     in.Stats.TodayActualCost,
			StandardCost:   in.Stats.TodayCost,
			RPM:            in.Stats.RPM,
			TPM:            in.Stats.TPM,
			AverageLatency: in.Stats.AverageDurationMs,
			TokenBreakdown: TokenBreakdown{
				Input:         in.Stats.TodayInputTokens,
				Output:        in.Stats.TodayOutputTokens,
				CacheCreation: in.Stats.TodayCacheCreationTokens,
				CacheRead:     in.Stats.TodayCacheReadTokens,
			},
		},
		Trend:  append([]sub2api.TrendDataPoint(nil), in.Trend...),
		Models: models,
	}
}
