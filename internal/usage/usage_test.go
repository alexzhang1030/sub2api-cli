package usage

import (
	"testing"
	"time"

	"github.com/alex/sub2api-cli/internal/sub2api"
)

func TestBuildTodayReportSortsModelsByActualCostThenTokens(t *testing.T) {
	report := BuildTodayReport(Input{
		Profile:   "work",
		Date:      "2026-05-07",
		Timezone:  "Asia/Shanghai",
		Generated: time.Date(2026, 5, 7, 13, 0, 0, 0, time.UTC),
		Stats: sub2api.DashboardStats{
			TodayTokens:              120,
			TodayActualCost:          2.5,
			TodayCost:                3,
			TodayRequests:            4,
			TodayInputTokens:         50,
			TodayOutputTokens:        40,
			TodayCacheCreationTokens: 10,
			TodayCacheReadTokens:     20,
		},
		Models: []sub2api.ModelStat{
			{Model: "small", TotalTokens: 1000, ActualCost: 1},
			{Model: "large", TotalTokens: 50, ActualCost: 9},
			{Model: "medium", TotalTokens: 2000, ActualCost: 1},
		},
		Trend: []sub2api.TrendDataPoint{
			{Date: "00:00", TotalTokens: 5, ActualCost: 0.1},
			{Date: "01:00", TotalTokens: 10, ActualCost: 0.2},
		},
	})

	if report.Models[0].Model != "large" {
		t.Fatalf("first model = %s", report.Models[0].Model)
	}
	if report.Models[1].Model != "medium" {
		t.Fatalf("second model = %s", report.Models[1].Model)
	}
	if report.Totals.TokenBreakdown.CacheRead != 20 {
		t.Fatalf("cache read = %d", report.Totals.TokenBreakdown.CacheRead)
	}
}
