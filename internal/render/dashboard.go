package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/alex/sub2api-cli/internal/sub2api"
	"github.com/alex/sub2api-cli/internal/usage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dashboardModel struct {
	report       usage.Report
	width        int
	refreshEvery time.Duration
	refresh      func() (usage.Report, error)
	refreshErr   string
}

func NewDashboard(report usage.Report) tea.Model {
	return dashboardModel{report: report, width: 100}
}

func NewLiveDashboard(report usage.Report, refreshEvery time.Duration, refresh func() (usage.Report, error)) tea.Model {
	return dashboardModel{
		report:       report,
		width:        100,
		refreshEvery: refreshEvery,
		refresh:      refresh,
	}
}

type refreshTickMsg struct{}

func (m dashboardModel) Init() tea.Cmd {
	return m.scheduleRefresh()
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
	case refreshTickMsg:
		if m.refresh == nil {
			return m, nil
		}
		report, err := m.refresh()
		if err != nil {
			m.refreshErr = err.Error()
		} else {
			m.report = report
			m.refreshErr = ""
		}
		return m, m.scheduleRefresh()
	case tea.KeyMsg:
		switch v.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m dashboardModel) View() string {
	view := Render(m.report, m.width)
	if strings.TrimSpace(m.refreshErr) != "" {
		view += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("refresh failed: "+m.refreshErr)
	}
	return view + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("press Enter to exit")
}

func (m dashboardModel) scheduleRefresh() tea.Cmd {
	if m.refresh == nil || m.refreshEvery <= 0 {
		return nil
	}
	return tea.Tick(m.refreshEvery, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func Render(report usage.Report, width int) string {
	if width <= 0 {
		width = 100
	}
	compact := width < 84
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("sub2api today")
	meta := fmt.Sprintf("%s  %s  %s", report.Profile, report.Date, report.Timezone)
	b.WriteString(title + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(meta+"  generated "+report.Generated.Format(time.RFC3339)) + "\n\n")
	if compact {
		b.WriteString(renderCompact(report))
	} else {
		b.WriteString(renderWide(report))
	}
	return b.String()
}

func renderWide(report usage.Report) string {
	card := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 2).Width(22)
	row := lipgloss.JoinHorizontal(lipgloss.Top,
		card.Render("Actual\n"+money(report.Totals.ActualCost)),
		card.Render("Standard\n"+money(report.Totals.StandardCost)),
		card.Render("Requests\n"+number(report.Totals.Requests)),
		card.Render("Tokens\n"+number(report.Totals.TotalTokens)),
	)
	return row + "\n\n" + renderBreakdown(report) + "\n\n" + renderTrend(report) + "\n\n" + renderModels(report)
}

func renderCompact(report usage.Report) string {
	return fmt.Sprintf("Actual %s | Standard %s | Requests %s | Tokens %s\n\n%s\n\n%s\n\n%s",
		money(report.Totals.ActualCost),
		money(report.Totals.StandardCost),
		number(report.Totals.Requests),
		number(report.Totals.TotalTokens),
		renderBreakdown(report),
		renderTrend(report),
		renderModels(report),
	)
}

func renderBreakdown(report usage.Report) string {
	t := report.Totals.TokenBreakdown
	return fmt.Sprintf("Tokens  input %s  output %s  cache write %s  cache read %s  rpm %s  tpm %s",
		number(t.Input), number(t.Output), number(t.CacheCreation), number(t.CacheRead), number(report.Totals.RPM), number(report.Totals.TPM))
}

func renderTrend(report usage.Report) string {
	values := make([]int64, 0, len(report.Trend))
	for _, p := range report.Trend {
		values = append(values, p.TotalTokens)
	}
	var b strings.Builder
	b.WriteString("Hourly token trend  " + Sparkline(values))
	limit := len(report.Trend)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		p := report.Trend[i]
		b.WriteString(fmt.Sprintf("\n%-12s %10s tokens  %8s", p.Date, number(p.TotalTokens), money(p.ActualCost)))
	}
	return b.String()
}

func renderModels(report usage.Report) string {
	var b strings.Builder
	b.WriteString("Model distribution")
	maxCost := 0.0
	for _, m := range report.Models {
		if m.ActualCost > maxCost {
			maxCost = m.ActualCost
		}
	}
	limit := len(report.Models)
	if limit > 8 {
		limit = 8
	}
	if limit == 0 {
		b.WriteString("\n(no model usage today)")
		return b.String()
	}
	for i := 0; i < limit; i++ {
		m := report.Models[i]
		b.WriteString(fmt.Sprintf("\n%-32s %10s tokens  %8s  %s", truncate(m.Model, 32), number(m.TotalTokens), money(m.ActualCost), bar(m.ActualCost, maxCost, 18)))
	}
	return b.String()
}

func bar(value, max float64, width int) string {
	if max <= 0 || width <= 0 {
		return ""
	}
	n := int(value / max * float64(width))
	if n == 0 && value > 0 {
		n = 1
	}
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

func money(v float64) string {
	return fmt.Sprintf("$%.4f", v)
}

func number(v int64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	raw := fmt.Sprintf("%d", v)
	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	return sign + strings.Join(parts, ",")
}

var _ sub2api.DashboardStats
