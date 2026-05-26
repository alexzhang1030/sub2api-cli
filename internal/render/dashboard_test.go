package render

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alex/sub2api-cli/internal/sub2api"
	"github.com/alex/sub2api-cli/internal/usage"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardRefreshTickUpdatesReportAndSchedulesNextRefresh(t *testing.T) {
	first := usage.Report{
		Profile:   "test",
		Date:      "2026-05-08",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
		Totals:    usage.Totals{TotalTokens: 10},
	}
	second := first
	second.Generated = time.Unix(2, 0)
	second.Totals.TotalTokens = 25

	calls := 0
	model := NewLiveDashboard(first, time.Hour, func() (usage.Report, error) {
		calls++
		return second, nil
	})

	updated, cmd := model.Update(refreshTickMsg{})
	if cmd == nil {
		t.Fatal("refresh tick did not schedule next refresh")
	}
	if calls != 1 {
		t.Fatalf("refresh calls = %d", calls)
	}
	if !strings.Contains(updated.View(), "25") {
		t.Fatalf("view after refresh = %q", updated.View())
	}
}

func TestDashboardRefreshTickKeepsPreviousReportOnError(t *testing.T) {
	report := usage.Report{
		Profile:   "test",
		Date:      "2026-05-08",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
		Totals:    usage.Totals{TotalTokens: 10},
	}
	model := NewLiveDashboard(report, time.Hour, func() (usage.Report, error) {
		return usage.Report{}, errors.New("boom")
	})

	updated, cmd := model.Update(refreshTickMsg{})
	if cmd == nil {
		t.Fatal("refresh tick did not schedule next refresh after error")
	}
	view := updated.View()
	if !strings.Contains(view, "10") {
		t.Fatalf("view after refresh error = %q", view)
	}
	if !strings.Contains(view, "boom") {
		t.Fatalf("view missing refresh error = %q", view)
	}
}

func TestDashboardFilterInputFiltersModels(t *testing.T) {
	report := usage.Report{
		Profile:   "test",
		Date:      "2026-05-08",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
		Totals: usage.Totals{
			Requests:    5,
			TotalTokens: 1150,
			TokenBreakdown: usage.TokenBreakdown{
				Input:         400,
				Output:        300,
				CacheCreation: 200,
				CacheRead:     250,
			},
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
			},
			{
				Model:               "claude-sonnet",
				Requests:            2,
				InputTokens:         300,
				OutputTokens:        260,
				CacheCreationTokens: 180,
				CacheReadTokens:     50,
				TotalTokens:         790,
			},
		},
	}
	model := NewDashboard(report)

	updated, cmd := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'f'}}))
	if cmd != nil {
		t.Fatal("filter key returned command")
	}
	updated, cmd = updated.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'g', 'p', 't'}}))
	if cmd != nil {
		t.Fatal("filter text returned command")
	}

	view := updated.View()
	if !strings.Contains(view, "filter gpt") {
		t.Fatalf("view missing active filter = %q", view)
	}
	if !strings.Contains(view, "openai/gpt-5.5") {
		t.Fatalf("view missing filtered model = %q", view)
	}
	if strings.Contains(view, "claude-sonnet") {
		t.Fatalf("view contains filtered out model = %q", view)
	}
	if !strings.Contains(view, "tokens 360") {
		t.Fatalf("view missing filtered token total = %q", view)
	}
}

func TestDashboardAllFilterShowsEveryModelInDetailTable(t *testing.T) {
	report := usage.Report{
		Profile:   "test",
		Date:      "2026-05-08",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
		Models: []sub2api.ModelStat{
			{Model: "openai/gpt-5.5", Requests: 3, TotalTokens: 360},
			{Model: "claude-sonnet", Requests: 2, TotalTokens: 790},
		},
	}
	model := NewDashboard(report)

	updated, _ := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'f'}}))
	updated, _ = updated.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'a', 'l', 'l'}}))

	view := updated.View()
	if !strings.Contains(view, "filter all") {
		t.Fatalf("view missing all filter = %q", view)
	}
	if !strings.Contains(view, "CACHE WRITE") {
		t.Fatalf("view missing detail table = %q", view)
	}
	if !strings.Contains(view, "openai/gpt-5.5") {
		t.Fatalf("view missing first model = %q", view)
	}
	if !strings.Contains(view, "claude-sonnet") {
		t.Fatalf("view missing second model = %q", view)
	}
}

func TestDashboardEscClearsActiveFilter(t *testing.T) {
	report := usage.Report{
		Profile:     "test",
		Date:        "2026-05-08",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Unix(1, 0),
		ModelFilter: "gpt",
		Models: []sub2api.ModelStat{
			{Model: "openai/gpt-5.5", TotalTokens: 360},
		},
	}
	model := NewDashboard(report)

	updated, cmd := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEsc}))
	if cmd != nil {
		t.Fatal("esc with active filter returned command")
	}

	view := updated.View()
	if strings.Contains(view, "model gpt") {
		t.Fatalf("view still contains model filter = %q", view)
	}
	if strings.Contains(view, "CACHE WRITE") {
		t.Fatalf("view still contains model detail table = %q", view)
	}
}

func TestDashboardBackspaceCanRestoreUnfilteredModels(t *testing.T) {
	report := usage.Report{
		Profile:   "test",
		Date:      "2026-05-08",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
		Models: []sub2api.ModelStat{
			{Model: "openai/gpt-5.5", TotalTokens: 360},
			{Model: "claude-sonnet", TotalTokens: 790},
		},
	}
	model := NewDashboard(report)

	updated, _ := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'f'}}))
	updated, _ = updated.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'g'}}))
	updated, _ = updated.Update(tea.KeyMsg(tea.Key{Type: tea.KeyBackspace}))

	view := updated.View()
	if !strings.Contains(view, "openai/gpt-5.5") {
		t.Fatalf("view missing first model after clearing filter = %q", view)
	}
	if !strings.Contains(view, "claude-sonnet") {
		t.Fatalf("view missing second model after clearing filter = %q", view)
	}
}

func TestDashboardRefreshPreservesActiveFilter(t *testing.T) {
	first := usage.Report{
		Profile:     "test",
		Date:        "2026-05-08",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Unix(1, 0),
		ModelFilter: "gpt",
		Models: []sub2api.ModelStat{
			{Model: "openai/gpt-5.5", TotalTokens: 360},
		},
	}
	second := first
	second.Generated = time.Unix(2, 0)
	second.Models = []sub2api.ModelStat{
		{Model: "openai/gpt-5.5", TotalTokens: 500},
		{Model: "claude-sonnet", TotalTokens: 700},
	}
	model := NewLiveDashboard(first, time.Hour, func() (usage.Report, error) {
		return second, nil
	})

	updated, cmd := model.Update(refreshTickMsg{})
	if cmd == nil {
		t.Fatal("refresh tick did not schedule next refresh")
	}

	view := updated.View()
	if !strings.Contains(view, "openai/gpt-5.5") {
		t.Fatalf("view missing filtered model after refresh = %q", view)
	}
	if strings.Contains(view, "claude-sonnet") {
		t.Fatalf("view contains filtered out model after refresh = %q", view)
	}
}

func TestRenderKeepsDefaultDashboardSummaryOnly(t *testing.T) {
	report := usage.Report{
		Profile:   "test",
		Date:      "2026-05-08",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
		Totals: usage.Totals{
			TotalTokens: 1000,
			TokenBreakdown: usage.TokenBreakdown{
				Input:         100,
				Output:        200,
				CacheCreation: 300,
				CacheRead:     400,
			},
		},
		Models: []sub2api.ModelStat{
			{
				Model:               "openai/gpt-5.5",
				InputTokens:         100,
				OutputTokens:        200,
				CacheCreationTokens: 300,
				CacheReadTokens:     400,
				TotalTokens:         1000,
			},
		},
	}

	view := Render(report, 100)

	if strings.Contains(view, "Standard") {
		t.Fatalf("view contains standard card = %q", view)
	}
	if !strings.Contains(view, "Cache Hit Rate") {
		t.Fatalf("view missing cache hit rate card = %q", view)
	}
	if !strings.Contains(view, "40.0%") {
		t.Fatalf("view missing cache hit rate value = %q", view)
	}
	actualIndex := strings.Index(view, "Actual")
	requestsIndex := strings.Index(view, "Requests")
	tokensIndex := strings.Index(view, "Tokens")
	cacheHitIndex := strings.Index(view, "Cache Hit Rate")
	if actualIndex < 0 || requestsIndex < 0 || tokensIndex < 0 || cacheHitIndex < 0 {
		t.Fatalf("view missing summary card labels = %q", view)
	}
	if !(actualIndex < requestsIndex && requestsIndex < tokensIndex && tokensIndex < cacheHitIndex) {
		t.Fatalf("summary cards out of order = %q", view)
	}
	if strings.Contains(view, "CACHE WRITE") {
		t.Fatalf("view contains model detail table = %q", view)
	}
	if strings.Contains(view, "in 100  out 200") {
		t.Fatalf("view contains model token details = %q", view)
	}
}

func TestRenderAllTimeTitle(t *testing.T) {
	report := usage.Report{
		Profile:   "test",
		Date:      "all-time",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
	}

	view := Render(report, 100)

	if !strings.Contains(view, "sub2api all-time") {
		t.Fatalf("view missing all-time title = %q", view)
	}
}

func TestRenderModelDistributionAlignsValuesAndShowsPercent(t *testing.T) {
	report := usage.Report{
		Profile:   "test",
		Date:      "all-time",
		Timezone:  "Asia/Shanghai",
		Generated: time.Unix(1, 0),
		Models: []sub2api.ModelStat{
			{Model: "gpt-5.5", TotalTokens: 1_198_700_000, ActualCost: 1070.9607},
			{Model: "gpt-5.4", TotalTokens: 289_300_000, ActualCost: 153.8055},
			{Model: "gpt-5.2", TotalTokens: 705_098, ActualCost: 0.6655},
		},
	}

	view := renderModels(report)
	lines := strings.Split(view, "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %#v", lines)
	}

	for _, want := range []string{"87.4%", "12.6%", "0.1%"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q = %q", want, view)
		}
	}
	tokenEnd := strings.Index(lines[1], " tokens")
	costEnd := strings.Index(lines[1], "$1070.9607") + len("$1070.9607")
	percentEnd := strings.Index(lines[1], "87.4%") + len("87.4%")
	if tokenEnd < 0 || costEnd < 0 || percentEnd < 0 {
		t.Fatalf("line 1 missing expected values = %q", lines[1])
	}
	for i, line := range lines[2:] {
		tokenValue := []string{"289.3M", "705,098"}[i]
		costValue := []string{"$153.8055", "$0.6655"}[i]
		percentValue := []string{"12.6%", "0.1%"}[i]
		if got := strings.Index(line, " tokens"); got != tokenEnd {
			t.Fatalf("line %d token column end = %d, want %d: %q", i+2, got, tokenEnd, line)
		}
		if got := strings.Index(line, costValue) + len(costValue); got != costEnd {
			t.Fatalf("line %d cost column end = %d, want %d: %q", i+2, got, costEnd, line)
		}
		if got := strings.Index(line, percentValue) + len(percentValue); got != percentEnd {
			t.Fatalf("line %d percent column end = %d, want %d: %q", i+2, got, percentEnd, line)
		}
		if !strings.Contains(line, tokenValue) || !strings.Contains(line, costValue) {
			t.Fatalf("line %d missing expected value: %q", i+2, line)
		}
	}
}

func TestRenderShowsZeroCacheHitRateInModelDetail(t *testing.T) {
	report := usage.Report{
		Profile:     "test",
		Date:        "2026-05-08",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Unix(1, 0),
		ModelFilter: "gpt-5.5",
	}

	view := Render(report, 72)

	if !strings.Contains(view, "cache hit 0.0%") {
		t.Fatalf("view missing zero cache hit rate = %q", view)
	}
}

func TestRenderShowsModelFilterDetailTable(t *testing.T) {
	report := usage.Report{
		Profile:     "test",
		Date:        "2026-05-08",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Unix(1, 0),
		ModelFilter: "gpt-5.5",
		Totals: usage.Totals{
			Requests:    3,
			TotalTokens: 360,
			TokenBreakdown: usage.TokenBreakdown{
				Input:         100,
				Output:        40,
				CacheCreation: 20,
				CacheRead:     200,
			},
		},
		Models: []sub2api.ModelStat{
			{
				Model:               "openai/gpt-5.5",
				InputTokens:         100,
				OutputTokens:        40,
				CacheCreationTokens: 20,
				CacheReadTokens:     200,
				TotalTokens:         360,
			},
		},
	}

	view := Render(report, 100)

	if !strings.Contains(view, "model gpt-5.5") {
		t.Fatalf("view missing model filter = %q", view)
	}
	if !strings.Contains(view, "MODEL") {
		t.Fatalf("view missing model table header = %q", view)
	}
	if !strings.Contains(view, "cache hit 55.6%") {
		t.Fatalf("view missing total-token cache hit rate = %q", view)
	}
	if !strings.Contains(view, "openai/gpt-5.5") {
		t.Fatalf("view missing model name = %q", view)
	}
	for _, want := range []string{"INPUT", "OUTPUT", "CACHE WRITE", "CACHE READ", "100", "40", "20", "200"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q = %q", want, view)
		}
	}
	if !strings.Contains(view, "TOTAL") {
		t.Fatalf("view missing total row = %q", view)
	}
}

func TestRenderModelDetailUsesPlainTableStyling(t *testing.T) {
	report := usage.Report{
		Profile:     "test",
		Date:        "2026-05-08",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Unix(1, 0),
		ModelFilter: "gpt",
		Models: []sub2api.ModelStat{
			{Model: "gpt-5.5", Requests: 189, TotalTokens: 18_673_706, InputTokens: 1_679_238, OutputTokens: 133_412, CacheReadTokens: 16_861_056, ActualCost: 20.8291},
		},
	}

	view := Render(report, 100)

	for _, border := range []string{"│", "─", "┌", "┐", "└", "┘"} {
		if strings.Contains(view, border) {
			t.Fatalf("view contains table border %q = %q", border, view)
		}
	}
	for _, color := range []string{"\x1b[38;5;39m", "\x1b[38;5;63m", "\x1b[38;5;214m"} {
		if strings.Contains(view, color) {
			t.Fatalf("view contains loud color %q = %q", color, view)
		}
	}
	if tableBodyColor != "252" {
		t.Fatalf("table body color = %s", tableBodyColor)
	}
	if tableTotalColor != "255" {
		t.Fatalf("table total color = %s", tableTotalColor)
	}
}

func TestRenderModelDetailAbbreviatesLargeTokenCounts(t *testing.T) {
	report := usage.Report{
		Profile:     "test",
		Date:        "2026-05-08",
		Timezone:    "Asia/Shanghai",
		Generated:   time.Unix(1, 0),
		ModelFilter: "gpt",
		Totals: usage.Totals{
			TotalTokens: 18_673_706,
			TokenBreakdown: usage.TokenBreakdown{
				Input:     1_679_238,
				Output:    133_412,
				CacheRead: 16_861_056,
			},
		},
		Models: []sub2api.ModelStat{
			{Model: "gpt-5.5", Requests: 189, TotalTokens: 18_673_706, InputTokens: 1_679_238, OutputTokens: 133_412, CacheReadTokens: 16_861_056, ActualCost: 20.8291},
		},
	}

	view := Render(report, 100)

	for _, want := range []string{"tokens 18.7M", "input 1.7M", "output 133,412", "cache read 16.9M", "18.7M", "1.7M", "16.9M"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q = %q", want, view)
		}
	}
	for _, tooLong := range []string{"18,673,706", "1,679,238", "16,861,056"} {
		if strings.Contains(view, tooLong) {
			t.Fatalf("view contains long token count %q = %q", tooLong, view)
		}
	}
	if !strings.Contains(view, "189") {
		t.Fatalf("view missing unabridged request count = %q", view)
	}
}

func TestDashboardOnlyExitKeysQuit(t *testing.T) {
	report := usage.Report{Profile: "test", Date: "2026-05-08", Timezone: "Asia/Shanghai"}
	model := NewLiveDashboard(report, time.Hour, nil)

	_, cmd := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'x'}}))
	if cmd != nil {
		t.Fatal("regular key triggered quit command")
	}

	_, cmd = model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("enter did not trigger quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("enter command is not tea.Quit")
	}
}
