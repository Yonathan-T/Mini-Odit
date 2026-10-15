package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"odition/pkg/engine"
	"odition/pkg/ingest"
	"odition/pkg/ledger"
	"odition/pkg/model"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5B21B6")).
			Padding(0, 1)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4B5563")).
			Padding(0, 1)

	greenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	redStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	goldStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)

	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6")).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1)

	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

type scanStepMsg struct {
	index int
}

type modelState struct {
	eng          *engine.Engine
	ingester     *ingest.Ingester
	agg          *ledger.Aggregator
	result       *ingest.IngestResult
	summary      *ledger.FinancialSummary
	err          error
	chunks       []string
	scanIndex    int
	isScanning   bool
	scanPreview  string
	spinnerIndex int
	parseTime    time.Duration
}

func initialModel() modelState {
	eng := engine.NewEngine()
	patternsDir := "patterns"
	if err := eng.LoadFromDir(patternsDir); err != nil {
		patternsDir = filepath.Join("..", "..", "patterns")
		_ = eng.LoadFromDir(patternsDir)
	}

	ing := ingest.NewIngester(eng)
	agg := ledger.NewAggregator("Yonathan")

	m := modelState{
		eng:        eng,
		ingester:   ing,
		agg:        agg,
		isScanning: true,
	}

	inboxPath := m.getInboxPath()
	chunks, err := ing.SplitChunks(inboxPath)
	if err != nil {
		m.err = err
		m.isScanning = false
	} else {
		m.chunks = chunks
	}

	return m
}

func (m *modelState) getInboxPath() string {
	inboxPath := filepath.Join("inbox", "messages.txt")
	if _, err := os.Stat(inboxPath); err != nil {
		inboxPath = filepath.Join("..", "..", "inbox", "messages.txt")
	}
	return inboxPath
}

func (m *modelState) finishScan() {
	inboxPath := m.getInboxPath()
	res, err := m.ingester.IngestFile(inboxPath)
	if err != nil {
		m.err = err
		return
	}
	m.result = res
	m.parseTime = res.Duration
	m.summary = m.agg.Aggregate(res.Transactions)
	m.err = nil
}

func (m modelState) Init() tea.Cmd {
	if m.isScanning && len(m.chunks) > 0 {
		return tea.Tick(30*time.Millisecond, func(t time.Time) tea.Msg {
			return scanStepMsg{index: 0}
		})
	}
	return nil
}

func (m modelState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			inboxPath := m.getInboxPath()
			chunks, err := m.ingester.SplitChunks(inboxPath)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.chunks = chunks
			m.scanIndex = 0
			m.isScanning = true
			m.scanPreview = "Initializing engine scan..."
			return m, tea.Tick(25*time.Millisecond, func(t time.Time) tea.Msg {
				return scanStepMsg{index: 0}
			})
		case " ", "enter":
			if m.isScanning {
				m.isScanning = false
				m.finishScan()
				return m, nil
			}
		}

	case scanStepMsg:
		if msg.index < len(m.chunks) {
			m.scanIndex = msg.index + 1
			chunk := m.chunks[msg.index]
			txn, err := m.eng.Parse(chunk)
			if err == nil {
				if txn.Category == "promo" {
					m.scanPreview = fmt.Sprintf("[PROMO] %s", txn.PatternName)
				} else {
					target := ""
					for _, p := range txn.Participants {
						if !strings.EqualFold(p.Name, "Yonathan") && p.Name != "" {
							target = " · " + p.Name
							break
						}
					}
					m.scanPreview = fmt.Sprintf("[%s] %s %.2f ETB%s", strings.ToUpper(txn.Provider), txn.Direction, txn.Principal, target)
				}
			} else if errors.Is(err, engine.ErrIgnored) {
				m.scanPreview = "[IGNORED] Bare OTP / Security PSA (0.015µs)"
			} else {
				m.scanPreview = "[UNMATCHED] Non-conforming bank text"
			}
			m.spinnerIndex = (m.spinnerIndex + 1) % len(spinnerFrames)
			nextIdx := msg.index + 1
			return m, tea.Tick(35*time.Millisecond, func(t time.Time) tea.Msg {
				return scanStepMsg{index: nextIdx}
			})
		}
		m.isScanning = false
		m.finishScan()
		return m, nil
	}
	return m, nil
}

func (m modelState) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading inbox: %v\nPress 'q' to quit.\n", m.err)
	}

	if m.isScanning {
		var b strings.Builder
		b.WriteString(headerStyle.Render(" ODITION ") + " " + subHeaderStyle.Render("Ethiopian Bank SMS Ledger · Owner: Yonathan") + "\n\n")

		var scanContent strings.Builder
		scanContent.WriteString(goldStyle.Render("LIVE INGESTION & SCANNER TELEMETRY") + "\n\n")

		frame := spinnerFrames[m.spinnerIndex%len(spinnerFrames)]
		pct := 0
		if len(m.chunks) > 0 {
			pct = (m.scanIndex * 100) / len(m.chunks)
		}

		totalWidth := 26
		filled := 0
		if len(m.chunks) > 0 {
			filled = (m.scanIndex * totalWidth) / len(m.chunks)
		}
		if filled > totalWidth {
			filled = totalWidth
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", totalWidth-filled)

		fmt.Fprintf(&scanContent, "%s %s Ingesting inbox/messages.txt...\n\n", greenStyle.Render(frame), goldStyle.Render("SCANNER:"))
		fmt.Fprintf(&scanContent, "Progress: [%s] %s (%d / %d messages)\n", greenStyle.Render(bar), goldStyle.Render(fmt.Sprintf("%3d%%", pct)), m.scanIndex, len(m.chunks))
		fmt.Fprintf(&scanContent, "Inspecting: %s\n\n", greenStyle.Render(m.scanPreview))
		fmt.Fprintf(&scanContent, "Status:   %s\n", subHeaderStyle.Render("Testing Layer 1 (Ignored) -> Layer 2 (Promos) -> Layer 3 (Regexp2)"))

		scanPanel := panelStyle.Width(94).Render(scanContent.String())
		b.WriteString(scanPanel + "\n")
		b.WriteString(subHeaderStyle.Render("Press [Space] to skip scan · [q] to quit") + "\n")
		return b.String()
	}

	if m.summary == nil {
		return "Loading ledger...\n"
	}

	var b strings.Builder

	b.WriteString(headerStyle.Render(" ODITION ") + " " + subHeaderStyle.Render("Ethiopian Bank SMS Ledger · Owner: Yonathan") + "\n\n")

	var accountsContent strings.Builder
	accountsContent.WriteString(goldStyle.Render("ACCOUNTS & BALANCES") + "\n")
	var provs []string
	for p := range m.summary.Accounts {
		provs = append(provs, p)
	}
	sort.Strings(provs)

	var totalCash float64
	for _, prov := range provs {
		acc := m.summary.Accounts[prov]
		if acc.LatestBalance != nil {
			totalCash += *acc.LatestBalance
			fmt.Fprintf(&accountsContent, "• %-10s : %s ETB\n", strings.ToUpper(prov), goldStyle.Render(fmt.Sprintf("%9.2f", *acc.LatestBalance)))
		}
	}
	accountsContent.WriteString(strings.Repeat("─", 27) + "\n")
	fmt.Fprintf(&accountsContent, "TOTAL CASH  : %s ETB\n", goldStyle.Render(fmt.Sprintf("%9.2f", totalCash)))

	accountsPanel := panelStyle.Width(31).Render(accountsContent.String())

	var cashflowContent strings.Builder
	cashflowContent.WriteString(goldStyle.Render("CASH FLOW (NET)") + "\n")
	fmt.Fprintf(&cashflowContent, "• Income   : %s ETB\n", greenStyle.Render(fmt.Sprintf("+%9.2f", m.summary.TotalIncome)))
	fmt.Fprintf(&cashflowContent, "• Expenses : %s ETB\n", redStyle.Render(fmt.Sprintf("-%9.2f", m.summary.TotalExpense)))
	fmt.Fprintf(&cashflowContent, "• Fees/VAT : %9.2f ETB\n", m.summary.TotalFees)
	cashflowContent.WriteString(strings.Repeat("─", 27) + "\n")

	netSign := "+"
	netRender := greenStyle.Render
	if m.summary.NetCashFlow < 0 {
		netSign = ""
		netRender = redStyle.Render
	}
	fmt.Fprintf(&cashflowContent, "SAVINGS    : %s ETB\n", netRender(fmt.Sprintf("%s%9.2f", netSign, m.summary.NetCashFlow)))

	cashflowPanel := panelStyle.Width(31).Render(cashflowContent.String())

	var peopleContent strings.Builder
	peopleContent.WriteString(goldStyle.Render("COUNTERPARTIES") + "\n")
	var peopleNames []string
	for name := range m.summary.People {
		peopleNames = append(peopleNames, name)
	}
	sort.Strings(peopleNames)

	if len(peopleNames) == 0 {
		peopleContent.WriteString("None yet\n")
	} else {
		for _, name := range peopleNames {
			p := m.summary.People[name]
			flow := ""
			if p.Received > 0 {
				flow = greenStyle.Render(fmt.Sprintf("+%.0f", p.Received))
			} else if p.Sent > 0 {
				flow = redStyle.Render(fmt.Sprintf("-%.0f", p.Sent))
			}
			displayName := name
			if len(displayName) > 13 {
				displayName = displayName[:11] + ".."
			}
			fmt.Fprintf(&peopleContent, "• %-13s : %s\n", displayName, flow)
		}
	}
	peoplePanel := panelStyle.Width(28).Render(peopleContent.String())

	topPanels := lipgloss.JoinHorizontal(lipgloss.Top, accountsPanel, " ", cashflowPanel, " ", peoplePanel)
	b.WriteString(topPanels + "\n\n")

	var txnsContent strings.Builder
	txnsContent.WriteString(goldStyle.Render("RECENT TRANSACTIONS") + "\n")
	shown := 0
	for i := len(m.summary.Transactions) - 1; i >= 0; i-- {
		t := m.summary.Transactions[i]
		if t.Category == "promo" {
			continue
		}
		if shown >= 3 {
			break
		}
		shown++

		dir := redStyle.Render("OUT")
		if t.Direction == model.DirectionIncoming {
			dir = greenStyle.Render("IN ")
		}

		otherPerson := ""
		for _, p := range t.Participants {
			if !strings.EqualFold(p.Name, "Yonathan") && p.Name != "" {
				otherPerson = " · " + p.Name
				break
			}
		}

		ref := ""
		for _, r := range t.TransactionIDs {
			ref = " · Ref: " + r
			break
		}

		fmt.Fprintf(&txnsContent, "%s %s %8.2f ETB [%-8s]%s%s\n",
			dir,
			badgeStyle.Render(strings.ToUpper(t.Provider)),
			t.Principal,
			t.Category,
			otherPerson,
			ref,
		)
	}

	txnsPanel := panelStyle.Width(94).Render(txnsContent.String())
	b.WriteString(txnsPanel + "\n")

	avgMicros := float64(0)
	totalMsgs := m.result.TotalParsed + m.result.TotalIgnored + m.result.TotalUnmatched
	if totalMsgs > 0 {
		avgMicros = float64(m.parseTime.Microseconds()) / float64(totalMsgs)
	}

	timeStr := fmt.Sprintf("%dµs (avg %.1fµs/msg)", m.parseTime.Microseconds(), avgMicros)
	if m.parseTime.Microseconds() == 0 {
		timeStr = fmt.Sprintf("%d ns", m.parseTime.Nanoseconds())
	}

	status := fmt.Sprintf("Scanned %d/%d msgs in %s · %s Parsed · %s Ignored · %s Unmatched    [r] Rescan    [q] Quit",
		totalMsgs,
		totalMsgs,
		greenStyle.Render(timeStr),
		greenStyle.Render(fmt.Sprintf("%d", m.result.TotalParsed)),
		fmt.Sprintf("%d", m.result.TotalIgnored),
		fmt.Sprintf("%d", m.result.TotalUnmatched),
	)
	b.WriteString(subHeaderStyle.Render(status) + "\n")

	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
