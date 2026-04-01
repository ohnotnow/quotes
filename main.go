package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const pageSize = 10

// --- Styles ---

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			PaddingBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			PaddingRight(2)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("170")).
			Bold(true)

	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242"))

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	formLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			PaddingTop(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// --- Modes ---

type mode int

const (
	modeList mode = iota
	modeSearch
	modeAdd
	modeView
	modeConfirmDelete
)

// --- Messages ---

type quotesLoadedMsg struct {
	quotes []Quote
	total  int
}

type quoteAddedMsg struct{}

// --- Model ---

type model struct {
	db          *sql.DB
	quotes      []Quote
	total       int
	page        int
	cursor      int
	mode        mode
	search      textinput.Model
	searchQuery string
	whoInput    textinput.Model
	bodyInput   textinput.Model
	addFocus      int // 0=who, 1=body
	deleteTarget  *Quote
	deleteConfirm bool // true=Yes selected, false=No selected
	flash         string
	width          int
	height         int
}

func initialModel(db *sql.DB) model {
	si := textinput.New()
	si.Placeholder = "Search quotes..."
	si.CharLimit = 100

	wi := textinput.New()
	wi.Placeholder = "Who said it?"
	wi.CharLimit = 200

	bi := textinput.New()
	bi.Placeholder = "The quote itself..."
	bi.CharLimit = 2000

	return model{
		db:       db,
		search:   si,
		whoInput: wi,
		bodyInput: bi,
		width:    80,
		height:   24,
	}
}

func (m model) loadQuotes() tea.Cmd {
	return func() tea.Msg {
		quotes, _ := fetchQuotes(m.db, m.searchQuery, pageSize, m.page*pageSize)
		total, _ := countQuotes(m.db, m.searchQuery)
		return quotesLoadedMsg{quotes: quotes, total: total}
	}
}

func (m model) Init() tea.Cmd {
	return m.loadQuotes()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case quotesLoadedMsg:
		m.quotes = msg.quotes
		m.total = msg.total
		return m, nil

	case quoteAddedMsg:
		m.mode = modeList
		m.flash = "Quote added!"
		m.page = 0
		m.cursor = 0
		m.searchQuery = ""
		return m, m.loadQuotes()

	case tea.KeyMsg:
		// Clear flash on any keypress
		m.flash = ""

		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeSearch:
			return m.updateSearch(msg)
		case modeAdd:
			return m.updateAdd(msg)
		case modeView:
			return m.updateView(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.quotes)-1 {
			m.cursor++
		}
	case "left", "h":
		if m.page > 0 {
			m.page--
			m.cursor = 0
			return m, m.loadQuotes()
		}
	case "right", "l":
		if (m.page+1)*pageSize < m.total {
			m.page++
			m.cursor = 0
			return m, m.loadQuotes()
		}
	case "/":
		m.mode = modeSearch
		m.search.SetValue(m.searchQuery)
		m.search.Focus()
		return m, textinput.Blink
	case "a":
		m.mode = modeAdd
		m.whoInput.SetValue("")
		m.bodyInput.SetValue("")
		m.addFocus = 0
		m.whoInput.Focus()
		m.bodyInput.Blur()
		return m, textinput.Blink
	case "d":
		if len(m.quotes) > 0 && m.cursor < len(m.quotes) {
			q := m.quotes[m.cursor]
			m.deleteTarget = &q
			m.deleteConfirm = false // default to "No"
			m.mode = modeConfirmDelete
		}
	case "enter":
		if len(m.quotes) > 0 && m.cursor < len(m.quotes) {
			m.mode = modeView
		}
	}
	return m, nil
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searchQuery = m.search.Value()
		m.mode = modeList
		m.page = 0
		m.cursor = 0
		m.search.Blur()
		return m, m.loadQuotes()
	case "esc":
		m.mode = modeList
		m.search.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

func (m model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		return m, nil
	case "tab", "shift+tab":
		if m.addFocus == 0 {
			m.addFocus = 1
			m.whoInput.Blur()
			m.bodyInput.Focus()
		} else {
			m.addFocus = 0
			m.bodyInput.Blur()
			m.whoInput.Focus()
		}
		return m, textinput.Blink
	case "enter":
		if m.addFocus == 0 {
			// Move to body field
			m.addFocus = 1
			m.whoInput.Blur()
			m.bodyInput.Focus()
			return m, textinput.Blink
		}
		// Submit
		who := strings.TrimSpace(m.whoInput.Value())
		body := strings.TrimSpace(m.bodyInput.Value())
		if who == "" || body == "" {
			m.flash = "Both fields are required"
			return m, nil
		}
		return m, func() tea.Msg {
			insertQuote(m.db, who, body)
			return quoteAddedMsg{}
		}
	}

	var cmd tea.Cmd
	if m.addFocus == 0 {
		m.whoInput, cmd = m.whoInput.Update(msg)
	} else {
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	}
	return m, cmd
}

func (m model) updateView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "enter":
		m.mode = modeList
	}
	return m, nil
}

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "right", "h", "l", "tab":
		m.deleteConfirm = !m.deleteConfirm
	case "y":
		m.deleteConfirm = true
		return m.executeDelete()
	case "n", "esc", "q":
		m.deleteTarget = nil
		m.mode = modeList
	case "enter":
		if m.deleteConfirm {
			return m.executeDelete()
		}
		m.deleteTarget = nil
		m.mode = modeList
	}
	return m, nil
}

func (m model) executeDelete() (tea.Model, tea.Cmd) {
	if m.deleteTarget != nil {
		deleteQuote(m.db, m.deleteTarget.ID)
		m.flash = fmt.Sprintf("Deleted quote #%d", m.deleteTarget.ID)
		m.deleteTarget = nil
		if m.cursor >= len(m.quotes)-1 && m.cursor > 0 {
			m.cursor--
		}
	}
	m.mode = modeList
	return m, m.loadQuotes()
}

func (m model) View() string {
	switch m.mode {
	case modeSearch:
		return m.viewSearch()
	case modeAdd:
		return m.viewAdd()
	case modeView:
		return m.viewDetail()
	case modeConfirmDelete:
		return m.viewConfirmDelete()
	default:
		return m.viewList()
	}
}

func (m model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📚 Quotes"))
	b.WriteString("\n")

	if m.searchQuery != "" {
		b.WriteString(searchStyle.Render(fmt.Sprintf("Search: %q", m.searchQuery)))
		b.WriteString("\n")
	}

	if m.flash != "" {
		b.WriteString(successStyle.Render(m.flash))
		b.WriteString("\n")
	}

	if len(m.quotes) == 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  No quotes found. Press 'a' to add one."))
		b.WriteString("\n")
	} else {
		// Column widths
		idW := 6
		whoW := 20
		bodyW := m.width - idW - whoW - 10
		if bodyW < 20 {
			bodyW = 20
		}

		// Header
		header := fmt.Sprintf("  %-*s %-*s %s", idW, "ID", whoW, "Who", "Quote")
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, idW+whoW+bodyW+10))))
		b.WriteString("\n")

		for i, q := range m.quotes {
			who := truncate(q.Who, whoW)
			body := truncate(q.Body, bodyW)
			line := fmt.Sprintf("  %-*d %-*s %s", idW, q.ID, whoW, who, body)

			if i == m.cursor {
				b.WriteString(selectedStyle.Render("▸ " + line[2:]))
			} else {
				b.WriteString(rowStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Pagination
	totalPages := (m.total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Page %d/%d (%d quotes)", m.page+1, totalPages, m.total)))
	b.WriteString("\n")

	// Help
	b.WriteString(helpStyle.Render("  ↑↓/jk navigate • ←→/hl page • / search • a add • d delete • enter view • q quit"))

	return b.String()
}

func (m model) viewConfirmDelete() string {
	if m.deleteTarget == nil {
		return ""
	}
	q := m.deleteTarget

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("170")).
		PaddingLeft(1)

	// Build the question
	body := truncate(q.Body, m.width-10)
	question := fmt.Sprintf("Delete '%s' by %s?", body, q.Who)

	// Buttons
	activeBtn := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("170")).
		Padding(0, 2).
		Bold(true)

	inactiveBtn := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("238")).
		Padding(0, 2)

	var yesBtn, noBtn string
	if m.deleteConfirm {
		yesBtn = activeBtn.Render("Yes")
		noBtn = inactiveBtn.Render("No")
	} else {
		yesBtn = inactiveBtn.Render("Yes")
		noBtn = activeBtn.Render("No")
	}
	buttons := yesBtn + "  " + noBtn

	var b strings.Builder
	b.WriteString(titleStyle.Render("🗑  Delete Quote"))
	b.WriteString("\n\n")
	b.WriteString(borderStyle.Render(question))
	b.WriteString("\n\n")
	b.WriteString("        " + buttons)
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  enter submit • y Yes • n No • ←→ toggle • esc cancel"))

	return b.String()
}

func (m model) viewSearch() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔍 Search Quotes"))
	b.WriteString("\n\n")
	b.WriteString("  " + m.search.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  enter to search • esc to cancel"))
	return b.String()
}

func (m model) viewAdd() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("✏️  Add Quote"))
	b.WriteString("\n\n")

	if m.flash != "" {
		b.WriteString("  " + successStyle.Render(m.flash))
		b.WriteString("\n\n")
	}

	b.WriteString("  " + formLabelStyle.Render("Who:"))
	b.WriteString("\n")
	b.WriteString("  " + m.whoInput.View())
	b.WriteString("\n\n")

	b.WriteString("  " + formLabelStyle.Render("Quote:"))
	b.WriteString("\n")
	b.WriteString("  " + m.bodyInput.View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("  tab to switch fields • enter on quote to save • esc to cancel"))

	return b.String()
}

func (m model) viewDetail() string {
	if m.cursor >= len(m.quotes) {
		return ""
	}
	q := m.quotes[m.cursor]

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Quote #%d", q.ID)))
	b.WriteString("\n\n")

	b.WriteString("  " + formLabelStyle.Render("Who: "))
	b.WriteString(rowStyle.Render(q.Who))
	b.WriteString("\n\n")

	// Word-wrap the body
	wrapped := wordWrap(q.Body, m.width-4)
	for _, line := range strings.Split(wrapped, "\n") {
		b.WriteString("  " + rowStyle.Render(line) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Added: %s", q.CreatedAt.Format("2 Jan 2006 15:04"))))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  esc/enter/q to go back"))

	return b.String()
}

// --- Helpers ---

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var result strings.Builder
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		lineLen := 0
		for i, w := range words {
			if i > 0 && lineLen+1+len(w) > width {
				result.WriteString("\n")
				lineLen = 0
			} else if i > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(w)
			lineLen += len(w)
		}
		result.WriteString("\n")
	}
	return strings.TrimRight(result.String(), "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- Main ---

func main() {
	db, err := openDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	p := tea.NewProgram(initialModel(db), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
