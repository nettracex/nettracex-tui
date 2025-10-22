// Package traceroute provides TUI model for traceroute operations
package traceroute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/tui"
)

// Model represents the traceroute TUI model
type Model struct {
	tool        *Tool
	state       ModelState
	host        string
	maxHops     int
	timeout     time.Duration
	packetSize  int
	queries     int
	ipv6        bool
	
	// UI components
	progress    progress.Model
	table       *tui.TableModel
	
	// Results
	hops        []domain.TraceHop
	statistics  TracerouteStatistics
	
	// State management
	ctx         context.Context
	cancel      context.CancelFunc
	resultChan  <-chan domain.TraceHop
	
	// UI state
	width       int
	height      int
	focused     bool
	
	// Real-time update tracking
	lastUpdate  time.Time
	updateCount int
	
	// Error handling
	err         error
	
	// Styles
	styles      ModelStyles
}

// ModelState represents the current state of the model
type ModelState int

const (
	StateInput ModelState = iota
	StateRunning
	StateCompleted
	StateError
)

// ModelStyles contains styling for the traceroute model
type ModelStyles struct {
	Base          lipgloss.Style
	Header        lipgloss.Style
	Table         lipgloss.Style
	Progress      lipgloss.Style
	Statistics    lipgloss.Style
	Error         lipgloss.Style
	Help          lipgloss.Style
	Focused       lipgloss.Style
	Blurred       lipgloss.Style
}

// NewModel creates a new traceroute model
func NewModel(tool *Tool) *Model {
	// Create progress bar
	p := progress.New(progress.WithDefaultGradient())

	// Create table with traceroute-specific headers
	headers := []string{"Hop", "Hostname", "IP Address", "RTT 1", "RTT 2", "RTT 3", "Status"}
	table := tui.NewTableModel(headers)

	m := &Model{
		tool:       tool,
		state:      StateInput,
		maxHops:    30,
		timeout:    5 * time.Second,
		packetSize: 60,
		queries:    3,
		ipv6:       false,
		progress:   p,
		table:      table,
		hops:       []domain.TraceHop{},
		styles:     NewModelStyles(),
	}

	return m
}

// NewModelStyles creates default styles for the traceroute model
func NewModelStyles() ModelStyles {
	return ModelStyles{
		Base: lipgloss.NewStyle().
			Padding(1, 2),
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Padding(0, 1),
		Table: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		Progress: lipgloss.NewStyle().
			Padding(0, 1),
		Statistics: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1).
			Margin(1, 0),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(1),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 0),
		Focused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")),
		Blurred: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTableSize()
		
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
			
		case "enter":
			if m.state == StateInput && m.host != "" {
				return m, m.startTraceroute()
			}
			
		case "r":
			if m.state == StateCompleted || m.state == StateError {
				m.reset()
				return m, nil
			}
			
		case "esc":
			if m.state == StateRunning && m.cancel != nil {
				m.cancel()
				m.state = StateInput
				return m, nil
			}
		}
		
	case StartTracerouteMsg:
		m.state = StateRunning
		return m, m.waitForNextHop()

	case HopReceivedMsg:
		m.hops = append(m.hops, msg.Hop)
		m.lastUpdate = time.Now()
		m.updateCount++
		m.updateTable()
		return m, m.waitForNextHop()
		
	case TracerouteCompleteMsg:
		m.state = StateCompleted
		m.statistics = m.tool.calculateStatistics(m.hops)
		return m, nil
		
	case TracerouteErrorMsg:
		m.err = msg.Error
		m.state = StateError
		return m, nil
	}

	// Update table component
	if m.table != nil {
		updatedTable, tableCmd := m.table.Update(msg)
		if tableModel, ok := updatedTable.(*tui.TableModel); ok {
			m.table = tableModel
		}
		cmds = append(cmds, tableCmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the model
func (m *Model) View() string {
	var sections []string

	// Header
	header := m.styles.Header.Render("NetTraceX - Traceroute")
	sections = append(sections, header)

	switch m.state {
	case StateInput:
		sections = append(sections, m.renderInputForm())
		
	case StateRunning:
		sections = append(sections, m.renderProgress())
		sections = append(sections, m.renderTable())
		
	case StateCompleted:
		sections = append(sections, m.renderTable())
		sections = append(sections, m.renderStatistics())
		
	case StateError:
		sections = append(sections, m.renderError())
	}

	// Help text
	sections = append(sections, m.renderHelp())

	return m.styles.Base.Render(strings.Join(sections, "\n\n"))
}

// SetSize sets the model size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.updateTableSize()
	if m.table != nil {
		m.table.SetSize(width-4, height-10) // Leave space for header, progress, and help
	}
}

// SetTheme sets the model theme
func (m *Model) SetTheme(theme domain.Theme) {
	// Update styles based on theme
	// This would be implemented based on the theme interface
}

// Focus focuses the model
func (m *Model) Focus() {
	m.focused = true
}

// Blur blurs the model
func (m *Model) Blur() {
	m.focused = false
}

// SetHost sets the target host for traceroute
func (m *Model) SetHost(host string) {
	m.host = host
}

// SetOptions sets traceroute options
func (m *Model) SetOptions(maxHops int, timeout time.Duration, packetSize int, queries int, ipv6 bool) {
	m.maxHops = maxHops
	m.timeout = timeout
	m.packetSize = packetSize
	m.queries = queries
	m.ipv6 = ipv6
}

// Custom messages for traceroute operations
type HopReceivedMsg struct {
	Hop domain.TraceHop
}

type TracerouteCompleteMsg struct{}

type TracerouteErrorMsg struct {
	Error error
}

type StartTracerouteMsg struct{}

// startTraceroute begins the traceroute operation
func (m *Model) startTraceroute() tea.Cmd {
	return func() tea.Msg {
		// Create context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		m.ctx = ctx
		m.cancel = cancel
		m.state = StateRunning

		// Get the result channel for real-time updates
		resultChan, err := m.tool.client.Traceroute(ctx, m.host, domain.TraceOptions{
			MaxHops:    m.maxHops,
			Timeout:    m.timeout,
			PacketSize: m.packetSize,
			Queries:    m.queries,
			IPv6:       m.ipv6,
		})
		if err != nil {
			return TracerouteErrorMsg{Error: err}
		}

		m.resultChan = resultChan
		return StartTracerouteMsg{}
	}
}

// waitForNextHop waits for the next hop result
func (m *Model) waitForNextHop() tea.Cmd {
	if m.resultChan == nil {
		// Return a no-op command instead of nil for testing
		return func() tea.Msg {
			return nil
		}
	}
	
	return func() tea.Msg {
		select {
		case hop, ok := <-m.resultChan:
			if !ok {
				return TracerouteCompleteMsg{}
			}
			return HopReceivedMsg{Hop: hop}
		case <-m.ctx.Done():
			return TracerouteErrorMsg{Error: m.ctx.Err()}
		}
	}
}

// reset resets the model to initial state
func (m *Model) reset() {
	if m.cancel != nil {
		m.cancel()
	}
	m.state = StateInput
	m.hops = []domain.TraceHop{}
	m.statistics = TracerouteStatistics{}
	m.err = nil
	m.ctx = nil
	m.cancel = nil
	m.resultChan = nil
	m.lastUpdate = time.Time{}
	m.updateCount = 0
	
	// Clear table data
	if m.table != nil {
		m.table.SetData([][]string{})
	}
}

// updateTableSize updates the display size based on available space
func (m *Model) updateTableSize() {
	if m.table != nil {
		// Reserve space for header, progress bar, statistics, and help text
		tableHeight := m.height - 10
		if tableHeight < 5 {
			tableHeight = 5
		}
		m.table.SetSize(m.width-4, tableHeight)
	}
}

// updateTable updates the display with current hop data
func (m *Model) updateTable() {
	if m.table == nil {
		return
	}

	// Convert hops to table rows
	var rows [][]string
	for _, hop := range m.hops {
		row := m.hopToTableRow(hop)
		rows = append(rows, row)
	}

	m.table.SetData(rows)
}

// renderInputForm renders the input form
func (m *Model) renderInputForm() string {
	var lines []string
	
	lines = append(lines, "Enter target host for traceroute:")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Host: %s", m.host))
	lines = append(lines, fmt.Sprintf("Max Hops: %d", m.maxHops))
	lines = append(lines, fmt.Sprintf("Timeout: %v", m.timeout))
	lines = append(lines, fmt.Sprintf("Packet Size: %d bytes", m.packetSize))
	lines = append(lines, fmt.Sprintf("Queries per hop: %d", m.queries))
	lines = append(lines, fmt.Sprintf("IPv6: %t", m.ipv6))
	lines = append(lines, "")
	lines = append(lines, "Press Enter to start traceroute")
	
	return strings.Join(lines, "\n")
}

// renderProgress renders the progress indicator
func (m *Model) renderProgress() string {
	if len(m.hops) == 0 {
		return m.styles.Progress.Render("Starting traceroute...")
	}
	
	progress := float64(len(m.hops)) / float64(m.maxHops)
	if progress > 1.0 {
		progress = 1.0
	}
	
	progressBar := m.progress.ViewAs(progress)
	status := fmt.Sprintf("Hop %d/%d", len(m.hops), m.maxHops)
	
	return m.styles.Progress.Render(fmt.Sprintf("%s\n%s", status, progressBar))
}

// renderTable renders the hops table
func (m *Model) renderTable() string {
	if m.table == nil {
		return "Table not initialized"
	}

	// Add real-time update indicator if actively receiving hops
	var tableContent string
	if m.state == StateRunning && time.Since(m.lastUpdate) < 2*time.Second {
		updateIndicator := m.styles.Progress.Render(fmt.Sprintf("● Live updates (%d hops received)", m.updateCount))
		tableContent = updateIndicator + "\n\n" + m.table.View()
	} else {
		tableContent = m.table.View()
	}

	return m.styles.Table.Render(tableContent)
}

// hopToTableRow converts a TraceHop to a table row
func (m *Model) hopToTableRow(hop domain.TraceHop) []string {
	var rtt1, rtt2, rtt3 string
	
	if hop.Timeout {
		rtt1, rtt2, rtt3 = "*", "*", "*"
	} else {
		rtts := []string{"", "", ""}
		for i, rtt := range hop.RTT {
			if i < 3 {
				rtts[i] = fmt.Sprintf("%.1f ms", float64(rtt.Nanoseconds())/1000000.0)
			}
		}
		rtt1, rtt2, rtt3 = rtts[0], rtts[1], rtts[2]
	}
	
	hostname := hop.Host.Hostname
	if hostname == "" {
		hostname = "-"
	}
	
	ipAddr := hop.Host.IPAddress.String()
	if ipAddr == "<nil>" || ipAddr == "" {
		ipAddr = "-"
	}
	
	status := "✓ OK"
	if hop.Timeout {
		status = "✗ Timeout"
	}
	
	return []string{
		fmt.Sprintf("%d", hop.Number),
		hostname,
		ipAddr,
		rtt1,
		rtt2,
		rtt3,
		status,
	}
}

// renderStatistics renders the traceroute statistics
func (m *Model) renderStatistics() string {
	stats := FormatTracerouteStatistics(m.statistics)
	return m.styles.Statistics.Render(stats)
}

// renderError renders error information
func (m *Model) renderError() string {
	errorText := fmt.Sprintf("Error: %v", m.err)
	return m.styles.Error.Render(errorText)
}

// renderHelp renders help text
func (m *Model) renderHelp() string {
	var help []string
	
	switch m.state {
	case StateInput:
		help = append(help, "Enter: Start traceroute • q: Quit")
	case StateRunning:
		help = append(help, "Esc: Cancel • q: Quit")
	case StateCompleted, StateError:
		help = append(help, "r: Reset • q: Quit")
	}
	
	return m.styles.Help.Render(strings.Join(help, " • "))
}