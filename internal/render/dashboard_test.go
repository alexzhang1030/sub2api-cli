package render

import (
	"errors"
	"strings"
	"testing"
	"time"

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
