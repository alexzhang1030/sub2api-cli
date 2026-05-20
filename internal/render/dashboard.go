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
	filterInput  string
	filtering    bool
	width        int
	refreshEvery time.Duration
	refresh      func() (usage.Report, error)
	refreshErr   string
}

const (
	tableHeaderColor = "244"
	tableBodyColor   = "252"
	tableTotalColor  = "255"
)

func NewDashboard(report usage.Report) tea.Model {
	return dashboardModel{report: usage.WithModelFilter(report, ""), filterInput: strings.TrimSpace(report.ModelFilter), width: 100}
}

func NewLiveDashboard(report usage.Report, refreshEvery time.Duration, refresh func() (usage.Report, error)) tea.Model {
	return dashboardModel{
		report:       usage.WithModelFilter(report, ""),
		filterInput:  strings.TrimSpace(report.ModelFilter),
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
			m.report = usage.WithModelFilter(report, "")
			m.refreshErr = ""
		}
		return m, m.scheduleRefresh()
	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(v)
		}
		switch v.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyEsc:
			if strings.TrimSpace(m.filterInput) != "" {
				m.filterInput = ""
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyRunes:
			if string(v.Runes) == "f" {
				m.filtering = true
				return m, nil
			}
		}
	}
	return m, nil
}

func (m dashboardModel) View() string {
	view := Render(usage.WithModelFilter(m.report, m.filterInput), m.width)
	if strings.TrimSpace(m.refreshErr) != "" {
		view += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("refresh failed: "+m.refreshErr)
	}
	help := "press f to filter, Enter to exit"
	if m.filtering {
		help = "filter " + m.filterInput + "  Enter apply, Esc clear, Backspace delete"
	}
	return view + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(help)
}

func (m dashboardModel) updateFilter(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEnter:
		m.filtering = false
	case tea.KeyEsc:
		m.filterInput = ""
		m.filtering = false
	case tea.KeyBackspace, tea.KeyCtrlH:
		runes := []rune(m.filterInput)
		if len(runes) > 0 {
			m.filterInput = string(runes[:len(runes)-1])
		}
	case tea.KeyRunes:
		m.filterInput += string(key.Runes)
	}
	return m, nil
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
	scope := "today"
	if report.Date == "all-time" {
		scope = "all-time"
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("sub2api " + scope)
	meta := fmt.Sprintf("%s  %s  %s", report.Profile, report.Date, report.Timezone)
	if strings.TrimSpace(report.ModelFilter) != "" {
		meta += "  model " + strings.TrimSpace(report.ModelFilter)
	}
	b.WriteString(title + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(meta+"  generated "+report.Generated.Format(time.RFC3339)) + "\n\n")
	if strings.TrimSpace(report.ModelFilter) != "" {
		b.WriteString(renderModelDetail(report))
		return b.String()
	}
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
		card.Render("Tokens\n"+tokenNumber(report.Totals.TotalTokens)),
	)
	return row + "\n\n" + renderBreakdown(report) + "\n\n" + renderTrend(report) + "\n\n" + renderModels(report)
}

func renderCompact(report usage.Report) string {
	return fmt.Sprintf("Actual %s | Standard %s | Requests %s | Tokens %s\n\n%s\n\n%s\n\n%s",
		money(report.Totals.ActualCost),
		money(report.Totals.StandardCost),
		number(report.Totals.Requests),
		tokenNumber(report.Totals.TotalTokens),
		renderBreakdown(report),
		renderTrend(report),
		renderModels(report),
	)
}

func renderBreakdown(report usage.Report) string {
	t := report.Totals.TokenBreakdown
	return fmt.Sprintf("Tokens  input %s  output %s  cache write %s  cache read %s  rpm %s  tpm %s",
		tokenNumber(t.Input), tokenNumber(t.Output), tokenNumber(t.CacheCreation), tokenNumber(t.CacheRead), number(report.Totals.RPM), tokenNumber(report.Totals.TPM))
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
		b.WriteString(fmt.Sprintf("\n%-12s %10s tokens  %8s", p.Date, tokenNumber(p.TotalTokens), money(p.ActualCost)))
	}
	return b.String()
}

func renderModels(report usage.Report) string {
	var b strings.Builder
	b.WriteString("Model distribution")
	maxCost := 0.0
	totalCost := 0.0
	for _, m := range report.Models {
		if m.ActualCost > maxCost {
			maxCost = m.ActualCost
		}
		totalCost += m.ActualCost
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
		b.WriteString("\n" + renderModelDistributionRow(m, totalCost, maxCost))
	}
	return b.String()
}

func renderModelDistributionRow(model sub2api.ModelStat, totalCost float64, maxCost float64) string {
	return fmt.Sprintf("%-32s  %12s tokens  %10s  %6s  %s",
		truncate(model.Model, 32),
		tokenNumber(model.TotalTokens),
		money(model.ActualCost),
		costPercent(model.ActualCost, totalCost),
		bar(model.ActualCost, maxCost, 18),
	)
}

func renderModelDetail(report usage.Report) string {
	var b strings.Builder
	t := report.Totals.TokenBreakdown
	b.WriteString(fmt.Sprintf("Matched models  %s\n", number(int64(len(report.Models)))))
	b.WriteString(fmt.Sprintf("TOTAL  requests %s  tokens %s  actual %s  standard %s\n", number(report.Totals.Requests), tokenNumber(report.Totals.TotalTokens), money(report.Totals.ActualCost), money(report.Totals.StandardCost)))
	b.WriteString(fmt.Sprintf("Tokens  input %s  output %s  cache write %s  cache read %s  cache hit %s\n\n",
		tokenNumber(t.Input), tokenNumber(t.Output), tokenNumber(t.CacheCreation), tokenNumber(t.CacheRead), percent(cacheHitRate(t))))

	if len(report.Models) == 0 {
		b.WriteString("(no matching model usage today)")
		return b.String()
	}

	headers := []string{"MODEL", "REQ", "TOKENS", "INPUT", "OUTPUT", "CACHE WRITE", "CACHE READ", "ACTUAL"}
	rows := make([][]string, 0, len(report.Models)+1)
	for _, m := range report.Models {
		rows = append(rows, []string{
			m.Model,
			number(m.Requests),
			tokenNumber(m.TotalTokens),
			tokenNumber(m.InputTokens),
			tokenNumber(m.OutputTokens),
			tokenNumber(m.CacheCreationTokens),
			tokenNumber(m.CacheReadTokens),
			money(m.ActualCost),
		})
	}
	rows = append(rows, []string{
		"TOTAL",
		number(report.Totals.Requests),
		tokenNumber(report.Totals.TotalTokens),
		tokenNumber(t.Input),
		tokenNumber(t.Output),
		tokenNumber(t.CacheCreation),
		tokenNumber(t.CacheRead),
		money(report.Totals.ActualCost),
	})

	b.WriteString(renderPlainTable(headers, rows))
	return b.String()
}

func renderPlainTable(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = lipgloss.Width(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && lipgloss.Width(cell) > widths[i] {
				widths[i] = lipgloss.Width(cell)
			}
		}
	}

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tableHeaderColor))
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tableBodyColor))
	totalStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tableTotalColor))

	var b strings.Builder
	b.WriteString(renderPlainTableRow(headers, widths, headerStyle))
	for i, row := range rows {
		style := bodyStyle
		if i == len(rows)-1 {
			style = totalStyle
		}
		b.WriteString("\n" + renderPlainTableRow(row, widths, style))
	}
	return b.String()
}

func renderPlainTableRow(row []string, widths []int, style lipgloss.Style) string {
	cells := make([]string, 0, len(widths))
	for i := range widths {
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		cells = append(cells, padRight(cell, widths[i]))
	}
	return style.Render(strings.Join(cells, "  "))
}

func padRight(s string, width int) string {
	padding := width - lipgloss.Width(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
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

func tokenNumber(v int64) string {
	if v <= -1_000_000 || v >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(v)/1_000_000)
	}
	return number(v)
}

func percent(v float64) string {
	return fmt.Sprintf("%.1f%%", v*100)
}

func costPercent(value, total float64) string {
	if total <= 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", value/total*100)
}

func cacheHitRate(t usage.TokenBreakdown) float64 {
	totalCacheTokens := t.CacheCreation + t.CacheRead
	if totalCacheTokens <= 0 {
		return 0
	}
	return float64(t.CacheRead) / float64(totalCacheTokens)
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
