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

func TestBuildTodayReportFiltersAndAggregatesModels(t *testing.T) {
	report := BuildTodayReport(Input{
		Profile:     "work",
		Date:        "2026-05-07",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Date(2026, 5, 7, 13, 0, 0, 0, time.UTC),
		ModelFilter: "GPT-5.5",
		Stats: sub2api.DashboardStats{
			TodayTokens:              999,
			TodayActualCost:          9.99,
			TodayCost:                12,
			TodayRequests:            99,
			TodayInputTokens:         400,
			TodayOutputTokens:        300,
			TodayCacheCreationTokens: 200,
			TodayCacheReadTokens:     100,
		},
		Models: []sub2api.ModelStat{
			{
				Model:               "openai/gpt-5.5",
				Requests:            3,
				InputTokens:         100,
				OutputTokens:        40,
				CacheCreationTokens: 20,
				CacheReadTokens:     200,
				TotalTokens:         360,
				Cost:                1.2,
				ActualCost:          0.7,
			},
			{
				Model:               "claude-sonnet",
				Requests:            8,
				InputTokens:         300,
				OutputTokens:        260,
				CacheCreationTokens: 180,
				CacheReadTokens:     50,
				TotalTokens:         790,
				Cost:                2.4,
				ActualCost:          1.9,
			},
		},
	})

	if len(report.Models) != 1 {
		t.Fatalf("models = %d", len(report.Models))
	}
	if report.Models[0].Model != "openai/gpt-5.5" {
		t.Fatalf("model = %s", report.Models[0].Model)
	}
	if report.Totals.Requests != 3 {
		t.Fatalf("requests = %d", report.Totals.Requests)
	}
	if report.Totals.TotalTokens != 360 {
		t.Fatalf("tokens = %d", report.Totals.TotalTokens)
	}
	if report.Totals.TokenBreakdown.Input != 100 {
		t.Fatalf("input = %d", report.Totals.TokenBreakdown.Input)
	}
	if report.Totals.TokenBreakdown.Output != 40 {
		t.Fatalf("output = %d", report.Totals.TokenBreakdown.Output)
	}
	if report.Totals.TokenBreakdown.CacheCreation != 20 {
		t.Fatalf("cache creation = %d", report.Totals.TokenBreakdown.CacheCreation)
	}
	if report.Totals.TokenBreakdown.CacheRead != 200 {
		t.Fatalf("cache read = %d", report.Totals.TokenBreakdown.CacheRead)
	}
}

func TestBuildTodayReportAllFilterShowsEveryModelInDetailMode(t *testing.T) {
	report := BuildTodayReport(Input{
		Profile:     "work",
		Date:        "2026-05-07",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Date(2026, 5, 7, 13, 0, 0, 0, time.UTC),
		ModelFilter: "all",
		Models: []sub2api.ModelStat{
			{Model: "openai/gpt-5.5", Requests: 3, TotalTokens: 360, InputTokens: 100},
			{Model: "claude-sonnet", Requests: 8, TotalTokens: 790, InputTokens: 300},
		},
	})

	if len(report.Models) != 2 {
		t.Fatalf("models = %d", len(report.Models))
	}
	if report.Totals.Requests != 11 {
		t.Fatalf("requests = %d", report.Totals.Requests)
	}
	if report.Totals.TotalTokens != 1150 {
		t.Fatalf("tokens = %d", report.Totals.TotalTokens)
	}
	if report.Totals.TokenBreakdown.Input != 400 {
		t.Fatalf("input = %d", report.Totals.TokenBreakdown.Input)
	}
}

func TestBuildAllReportUsesTotalStats(t *testing.T) {
	report := BuildAllReport(Input{
		Profile:   "work",
		Date:      "all-time",
		Timezone:  "Asia/Shanghai",
		Generated: time.Date(2026, 5, 7, 13, 0, 0, 0, time.UTC),
		Stats: sub2api.DashboardStats{
			TotalRequests:            30,
			TotalTokens:              3000,
			TotalActualCost:          4.5,
			TotalCost:                5.5,
			TotalInputTokens:         1100,
			TotalOutputTokens:        900,
			TotalCacheCreationTokens: 400,
			TotalCacheReadTokens:     600,
			TodayRequests:            3,
			TodayTokens:              300,
			TodayActualCost:          0.45,
			TodayCost:                0.55,
			TodayInputTokens:         110,
			TodayOutputTokens:        90,
			TodayCacheCreationTokens: 40,
			TodayCacheReadTokens:     60,
		},
		Models: []sub2api.ModelStat{
			{Model: "small", TotalTokens: 1000, ActualCost: 1},
			{Model: "large", TotalTokens: 50, ActualCost: 9},
		},
	})

	if report.Date != "all-time" {
		t.Fatalf("date = %q", report.Date)
	}
	if report.Totals.Requests != 30 {
		t.Fatalf("requests = %d", report.Totals.Requests)
	}
	if report.Totals.TotalTokens != 3000 {
		t.Fatalf("tokens = %d", report.Totals.TotalTokens)
	}
	if report.Totals.TokenBreakdown.Input != 1100 {
		t.Fatalf("input = %d", report.Totals.TokenBreakdown.Input)
	}
	if report.Totals.TokenBreakdown.Output != 900 {
		t.Fatalf("output = %d", report.Totals.TokenBreakdown.Output)
	}
	if report.Totals.TokenBreakdown.CacheCreation != 400 {
		t.Fatalf("cache creation = %d", report.Totals.TokenBreakdown.CacheCreation)
	}
	if report.Totals.TokenBreakdown.CacheRead != 600 {
		t.Fatalf("cache read = %d", report.Totals.TokenBreakdown.CacheRead)
	}
}
