// Package tui contains result view models for displaying diagnostic results
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// ResultViewMode represents different ways to display results
type ResultViewMode int

const (
	ResultViewModeFormatted ResultViewMode = iota
	ResultViewModeTable
	ResultViewModeRaw
)

// ResultViewModel handles display of diagnostic results
type ResultViewModel struct {
	result     domain.Result
	mode       ResultViewMode
	tableModel *TableModel
	width      int
	height     int
	theme      domain.Theme
	keyMap     KeyMap
	focused    bool
	scrollPager *StandardScrollPager // Migrated to StandardScrollPager for consistency
}

// NewResultViewModel creates a new result view model
func NewResultViewModel() *ResultViewModel {
	scrollPager := NewStandardScrollPager()
	scrollPager.SetShowScrollIndicators(true)
	
	return &ResultViewModel{
		mode:        ResultViewModeFormatted,
		tableModel:  NewTableModel([]string{}),
		keyMap:      DefaultKeyMap(),
		focused:     true,
		scrollPager: scrollPager,
	}
}

// Init implements tea.Model
func (m *ResultViewModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *ResultViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Update scroll pager for non-table modes
	if m.mode != ResultViewModeTable && m.scrollPager != nil {
		updatedModel, scrollCmd := m.scrollPager.Update(msg)
		if pager, ok := updatedModel.(*StandardScrollPager); ok {
			m.scrollPager = pager
			cmd = scrollCmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keyMap.Tab):
			// Cycle through view modes
			m.cycleViewMode()
			return m, cmd

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			// Switch to raw mode
			m.mode = ResultViewModeRaw
			return m, cmd

		case key.Matches(msg, key.NewBinding(key.WithKeys("f"))):
			// Switch to formatted mode
			m.mode = ResultViewModeFormatted
			return m, cmd

		case key.Matches(msg, key.NewBinding(key.WithKeys("t"))):
			// Switch to table mode
			m.mode = ResultViewModeTable
			return m, cmd
		}

		// Pass through to table model if in table mode
		if m.mode == ResultViewModeTable && m.tableModel != nil {
			updatedTable, tableCmd := m.tableModel.Update(msg)
			m.tableModel = updatedTable.(*TableModel)
			cmd = tableCmd
		}
	}

	return m, cmd
}

// View implements tea.Model
func (m *ResultViewModel) View() string {
	if m.result == nil {
		return m.renderNoResult()
	}

	// For table mode, use the existing table view (it has its own scrolling)
	if m.mode == ResultViewModeTable {
		var content strings.Builder
		content.WriteString(m.renderModeIndicator())
		content.WriteString("\n\n")
		content.WriteString(m.renderTableResult())
		content.WriteString("\n\n")
		content.WriteString(m.renderViewModeHelp())
		return content.String()
	}

	// For formatted and raw modes, use pager
	var mainContent strings.Builder
	
	// Only put the main result content in the pager
	switch m.mode {
	case ResultViewModeFormatted:
		mainContent.WriteString(m.renderFormattedResult())
	case ResultViewModeRaw:
		mainContent.WriteString(m.renderRawResult())
	}

	// Build the full view with header, scroll pager content, and footer
	var fullView strings.Builder
	fullView.WriteString(m.renderModeIndicator())
	fullView.WriteString("\n\n")

	// Convert content to scrollable items and set in scroll pager
	if m.scrollPager != nil {
		// Split content into lines and create scrollable items
		lines := strings.Split(mainContent.String(), "\n")
		items := make([]ScrollableItem, len(lines))
		for i, line := range lines {
			items[i] = NewStringScrollableItem(line, fmt.Sprintf("line_%d", i))
		}
		m.scrollPager.SetItems(items)
		fullView.WriteString(m.scrollPager.View())
	} else {
		fullView.WriteString(mainContent.String())
	}

	// View mode help
	fullView.WriteString("\n\n")
	fullView.WriteString(m.renderViewModeHelp())

	return fullView.String()
}

// SetResult sets the result to display
func (m *ResultViewModel) SetResult(result domain.Result) {
	m.result = result
	m.updateTableModel()
}

// renderNoResult renders a message when no result is available
func (m *ResultViewModel) renderNoResult() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	return style.Render("No result available")
}

// renderModeIndicator renders the current view mode indicator
func (m *ResultViewModel) renderModeIndicator() string {
	var modeText string
	switch m.mode {
	case ResultViewModeFormatted:
		modeText = "📋 Formatted View"
	case ResultViewModeTable:
		modeText = "📊 Table View"
	case ResultViewModeRaw:
		modeText = "📄 Raw Data View"
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	return style.Render(modeText)
}

// renderFormattedResult renders the result in formatted view
func (m *ResultViewModel) renderFormattedResult() string {
	if m.result == nil {
		return "No result to display"
	}

	// Handle different result types
	switch data := m.result.Data().(type) {
	case domain.WHOISResult:
		return m.renderWHOISResult(data)
	case []domain.PingResult:
		return m.renderPingResults(data)
	case domain.PingResult:
		return m.renderPingResult(data)
	case domain.DNSResult:
		return m.renderDNSResult(data)
	case domain.SSLResult:
		return m.renderSSLResult(data)
	case []domain.TraceHop:
		return m.renderTracerouteResults(data)
	case domain.TraceHop:
		return m.renderTraceHopResult(data)
	default:
		return fmt.Sprintf("Unsupported result type: %T", data)
	}
}

// renderWHOISResult renders WHOIS results in formatted view
func (m *ResultViewModel) renderWHOISResult(result domain.WHOISResult) string {
	var content strings.Builder

	// Domain information section
	content.WriteString(m.renderSection("Domain Information", [][]string{
		{"Domain", result.Domain},
		{"Registrar", result.Registrar},
	}))

	// Important dates section
	if !result.Created.IsZero() || !result.Updated.IsZero() || !result.Expires.IsZero() {
		dateInfo := [][]string{}
		if !result.Created.IsZero() {
			dateInfo = append(dateInfo, []string{"Created", result.Created.Format("2006-01-02 15:04:05")})
		}
		if !result.Updated.IsZero() {
			dateInfo = append(dateInfo, []string{"Updated", result.Updated.Format("2006-01-02 15:04:05")})
		}
		if !result.Expires.IsZero() {
			dateInfo = append(dateInfo, []string{"Expires", result.Expires.Format("2006-01-02 15:04:05")})
		}

		content.WriteString("\n")
		content.WriteString(m.renderSection("Important Dates", dateInfo))
	}

	// Name servers section
	if len(result.NameServers) > 0 {
		content.WriteString("\n")
		nsInfo := [][]string{}
		for i, ns := range result.NameServers {
			nsInfo = append(nsInfo, []string{fmt.Sprintf("NS %d", i+1), ns})
		}
		content.WriteString(m.renderSection("Name Servers", nsInfo))
	}

	// Status section
	if len(result.Status) > 0 {
		content.WriteString("\n")
		statusInfo := [][]string{}
		for i, status := range result.Status {
			statusInfo = append(statusInfo, []string{fmt.Sprintf("Status %d", i+1), status})
		}
		content.WriteString(m.renderSection("Domain Status", statusInfo))
	}

	// Contacts section
	if len(result.Contacts) > 0 {
		content.WriteString("\n")
		content.WriteString(m.renderContactsSection(result.Contacts))
	}

	return content.String()
}

// renderPingResults renders multiple ping results with statistics
func (m *ResultViewModel) renderPingResults(results []domain.PingResult) string {
	var content strings.Builder

	if len(results) == 0 {
		return "No ping results available"
	}

	// Get statistics from metadata if available
	var stats interface{}
	if m.result != nil {
		stats = m.result.Metadata()["statistics"]
	}

	// Summary section
	content.WriteString(m.renderSection("Ping Summary", [][]string{
		{"Target Host", results[0].Host.Hostname},
		{"Target IP", results[0].Host.IPAddress.String()},
		{"Total Pings", fmt.Sprintf("%d", len(results))},
	}))

	// Individual results (show last 5 for brevity)
	content.WriteString("\n")
	maxDisplay := 5
	startIdx := 0
	if len(results) > maxDisplay {
		startIdx = len(results) - maxDisplay
		content.WriteString(fmt.Sprintf("Recent Results (showing last %d of %d):\n", maxDisplay, len(results)))
	} else {
		content.WriteString("Ping Results:\n")
	}

	for i := startIdx; i < len(results); i++ {
		result := results[i]
		var status string
		if result.Error != nil {
			status = fmt.Sprintf("❌ Seq %d: %v", result.Sequence, result.Error)
		} else {
			status = fmt.Sprintf("✅ Seq %d: time=%v ttl=%d", result.Sequence, result.RTT, result.TTL)
		}
		content.WriteString("  " + status + "\n")
	}

	// Statistics if available
	if stats != nil {
		content.WriteString("\n")
		content.WriteString(m.renderSection("Statistics", m.formatPingStatistics(stats)))
	}

	return content.String()
}

// renderPingResult renders ping results (placeholder)
func (m *ResultViewModel) renderPingResult(result domain.PingResult) string {
	return m.renderSection("Ping Result", [][]string{
		{"Host", result.Host.Hostname},
		{"IP", result.Host.IPAddress.String()},
		{"Sequence", fmt.Sprintf("%d", result.Sequence)},
		{"RTT", result.RTT.String()},
		{"TTL", fmt.Sprintf("%d", result.TTL)},
		{"Timestamp", result.Timestamp.Format("2006-01-02 15:04:05")},
	})
}

// renderDNSResult renders DNS results with proper formatting and grouping
func (m *ResultViewModel) renderDNSResult(result domain.DNSResult) string {
	var content strings.Builder

	// Query information section
	content.WriteString(m.renderSection("DNS Query Information", [][]string{
		{"Domain", result.Query},
		{"Server", result.Server},
		{"Response Time", result.ResponseTime.String()},
		{"Total Records", fmt.Sprintf("%d", len(result.Records))},
	}))

	if len(result.Records) > 0 {
		// Group records by type for better display
		recordsByType := make(map[domain.DNSRecordType][]domain.DNSRecord)
		for _, record := range result.Records {
			recordsByType[record.Type] = append(recordsByType[record.Type], record)
		}

		// Display records grouped by type
		recordTypes := []domain.DNSRecordType{
			domain.DNSRecordTypeA,
			domain.DNSRecordTypeAAAA,
			domain.DNSRecordTypeMX,
			domain.DNSRecordTypeTXT,
			domain.DNSRecordTypeCNAME,
			domain.DNSRecordTypeNS,
			domain.DNSRecordTypeSOA,
			domain.DNSRecordTypePTR,
		}

		for _, recordType := range recordTypes {
			if records, exists := recordsByType[recordType]; exists && len(records) > 0 {
				content.WriteString("\n")
				
				// Create section for this record type
				recordTypeStr := m.getDNSRecordTypeString(recordType)
				sectionTitle := fmt.Sprintf("%s Records (%d)", recordTypeStr, len(records))
				
				recordInfo := [][]string{}
				for _, record := range records {
					if record.Priority > 0 {
						// For MX records, show priority
						recordInfo = append(recordInfo, []string{
							record.Name,
							fmt.Sprintf("%s (Priority: %d)", record.Value, record.Priority),
						})
					} else {
						recordInfo = append(recordInfo, []string{record.Name, record.Value})
					}
				}
				content.WriteString(m.renderSection(sectionTitle, recordInfo))
			}
		}
	}

	// Authority records section
	if len(result.Authority) > 0 {
		content.WriteString("\n")
		authorityInfo := [][]string{}
		for _, record := range result.Authority {
			authorityInfo = append(authorityInfo, []string{
				record.Name,
				fmt.Sprintf("%s (%s)", record.Value, m.getDNSRecordTypeString(record.Type)),
			})
		}
		content.WriteString(m.renderSection("Authority Records", authorityInfo))
	}

	// Additional records section
	if len(result.Additional) > 0 {
		content.WriteString("\n")
		additionalInfo := [][]string{}
		for _, record := range result.Additional {
			additionalInfo = append(additionalInfo, []string{
				record.Name,
				fmt.Sprintf("%s (%s)", record.Value, m.getDNSRecordTypeString(record.Type)),
			})
		}
		content.WriteString(m.renderSection("Additional Records", additionalInfo))
	}

	return content.String()
}

// renderSSLResult renders SSL results (placeholder)
func (m *ResultViewModel) renderSSLResult(result domain.SSLResult) string {
	return m.renderSection("SSL Certificate", [][]string{
		{"Host", result.Host},
		{"Port", fmt.Sprintf("%d", result.Port)},
		{"Valid", fmt.Sprintf("%t", result.Valid)},
		{"Issuer", result.Issuer},
		{"Subject", result.Subject},
		{"Expiry", result.Expiry.Format("2006-01-02 15:04:05")},
	})
}

// renderTracerouteResults renders multiple traceroute hop results
func (m *ResultViewModel) renderTracerouteResults(results []domain.TraceHop) string {
	var content strings.Builder

	if len(results) == 0 {
		return "No traceroute results available"
	}

	// Get statistics from metadata if available
	var stats interface{}
	if m.result != nil {
		stats = m.result.Metadata()["statistics"]
	}

	// Summary section - get target host from metadata, not from first hop
	targetHost := "Unknown"
	targetIP := "Unknown"
	if m.result != nil {
		if host, ok := m.result.Metadata()["host"].(string); ok && host != "" {
			targetHost = host
			targetIP = host // Use the same value for IP if we don't have separate IP info
		}
		// Try to get separate IP if available
		if ip, ok := m.result.Metadata()["target_ip"].(string); ok && ip != "" {
			targetIP = ip
		}
	}

	content.WriteString(m.renderSection("Traceroute Summary", [][]string{
		{"Target Host", targetHost},
		{"Target IP", targetIP},
		{"Total Hops", fmt.Sprintf("%d", len(results))},
	}))

	// Hop results
	content.WriteString("\n")
	content.WriteString("Traceroute Path:\n")

	for _, hop := range results {
		var status string
		var rttInfo string
		
		if hop.Timeout {
			status = "❌ Timeout"
			rttInfo = "* * *"
		} else {
			status = "✅ OK"
			if len(hop.RTT) > 0 {
				var rttStrs []string
				for _, rtt := range hop.RTT {
					rttStrs = append(rttStrs, fmt.Sprintf("%.1fms", float64(rtt.Nanoseconds())/1000000.0))
				}
				rttInfo = strings.Join(rttStrs, " ")
			} else {
				rttInfo = "No RTT data"
			}
		}

		hostname := hop.Host.Hostname
		ipAddr := ""
		if hop.Host.IPAddress != nil {
			ipAddr = hop.Host.IPAddress.String()
		}
		
		// If we don't have a hostname, use the IP address or show timeout indicator
		if hostname == "" {
			if ipAddr != "" {
				hostname = ipAddr
			} else if hop.Timeout {
				hostname = "*"
				ipAddr = "*"
			} else {
				hostname = "Unknown"
				ipAddr = "Unknown"
			}
		}
		
		// If we still don't have an IP address, use placeholder
		if ipAddr == "" {
			if hop.Timeout {
				ipAddr = "*"
			} else {
				ipAddr = "Unknown"
			}
		}

		hopLine := fmt.Sprintf("  %2d  %-20s %-15s %s  %s", 
			hop.Number, hostname, ipAddr, rttInfo, status)
		content.WriteString(hopLine + "\n")
	}

	// Statistics if available
	if stats != nil {
		content.WriteString("\n")
		content.WriteString(m.renderSection("Statistics", m.formatTracerouteStatistics(stats)))
	}

	return content.String()
}

// renderTraceHopResult renders traceroute hop results (placeholder)
func (m *ResultViewModel) renderTraceHopResult(result domain.TraceHop) string {
	return m.renderSection("Traceroute Hop", [][]string{
		{"Hop Number", fmt.Sprintf("%d", result.Number)},
		{"Host", result.Host.Hostname},
		{"IP", result.Host.IPAddress.String()},
		{"Timeout", fmt.Sprintf("%t", result.Timeout)},
		{"Timestamp", result.Timestamp.Format("2006-01-02 15:04:05")},
	})
}

// renderSection renders a section with key-value pairs
func (m *ResultViewModel) renderSection(title string, data [][]string) string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Width(15).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")

	for _, row := range data {
		if len(row) >= 2 && row[1] != "" {
			content.WriteString(keyStyle.Render(row[0]+":"))
			content.WriteString(" ")
			content.WriteString(valueStyle.Render(row[1]))
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderContactsSection renders the contacts section for WHOIS results
func (m *ResultViewModel) renderContactsSection(contacts map[string]domain.Contact) string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	content.WriteString(titleStyle.Render("Contacts"))
	content.WriteString("\n")

	for contactType, contact := range contacts {
		if contact.Name != "" || contact.Email != "" || contact.Organization != "" {
			contactData := [][]string{}

			if contact.Name != "" {
				contactData = append(contactData, []string{"Name", contact.Name})
			}
			if contact.Organization != "" {
				contactData = append(contactData, []string{"Organization", contact.Organization})
			}
			if contact.Email != "" {
				contactData = append(contactData, []string{"Email", contact.Email})
			}
			if contact.Phone != "" {
				contactData = append(contactData, []string{"Phone", contact.Phone})
			}

			if len(contactData) > 0 {
				content.WriteString(m.renderSection(strings.Title(contactType), contactData))
				content.WriteString("\n")
			}
		}
	}

	return content.String()
}

// renderTableResult renders the result in table view
func (m *ResultViewModel) renderTableResult() string {
	if m.tableModel == nil {
		return "Table view not available"
	}

	return m.tableModel.View()
}

// renderRawResult renders the result in raw view
func (m *ResultViewModel) renderRawResult() string {
	if m.result == nil {
		return "No raw data available"
	}

	// Export as JSON for raw view
	rawData, err := m.result.Export(domain.ExportFormatJSON)
	if err != nil {
		return fmt.Sprintf("Error exporting raw data: %v", err)
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(1).
		Width(m.width - 4)

	return style.Render(string(rawData))
}

// renderViewModeHelp renders help text for view modes
func (m *ResultViewModel) renderViewModeHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	var help string
	if m.mode == ResultViewModeTable {
		help = "f: formatted • t: table • r: raw • tab: cycle modes • ↑/↓: navigate table"
	} else {
		help = "f: formatted • t: table • r: raw • tab: cycle modes • ↑/↓: scroll • PgUp/PgDown: page • Home/End: jump"
	}
	return helpStyle.Render(help)
}

// cycleViewMode cycles through available view modes
func (m *ResultViewModel) cycleViewMode() {
	switch m.mode {
	case ResultViewModeFormatted:
		m.mode = ResultViewModeTable
	case ResultViewModeTable:
		m.mode = ResultViewModeRaw
	case ResultViewModeRaw:
		m.mode = ResultViewModeFormatted
	}
}

// updateTableModel updates the table model based on the current result
func (m *ResultViewModel) updateTableModel() {
	if m.result == nil {
		return
	}

	// Create table data based on result type
	switch data := m.result.Data().(type) {
	case domain.WHOISResult:
		m.updateWHOISTable(data)
	case []domain.PingResult:
		m.updatePingTable(data)
	case domain.DNSResult:
		m.updateDNSTable(data)
	case []domain.TraceHop:
		m.updateTracerouteTable(data)
	default:
		// Generic table for other types
		m.updateGenericTable()
	}
}

// updateWHOISTable updates table model for WHOIS results
func (m *ResultViewModel) updateWHOISTable(result domain.WHOISResult) {
	headers := []string{"Property", "Value"}
	m.tableModel = NewTableModel(headers)

	// Add basic information
	m.tableModel.AddRow([]string{"Domain", result.Domain})
	m.tableModel.AddRow([]string{"Registrar", result.Registrar})

	if !result.Created.IsZero() {
		m.tableModel.AddRow([]string{"Created", result.Created.Format("2006-01-02")})
	}
	if !result.Expires.IsZero() {
		m.tableModel.AddRow([]string{"Expires", result.Expires.Format("2006-01-02")})
	}

	// Add name servers
	for i, ns := range result.NameServers {
		m.tableModel.AddRow([]string{fmt.Sprintf("Name Server %d", i+1), ns})
	}
}

// updatePingTable updates table model for ping results
func (m *ResultViewModel) updatePingTable(results []domain.PingResult) {
	headers := []string{"Sequence", "Host", "IP", "RTT", "TTL", "Status"}
	m.tableModel = NewTableModel(headers)

	for _, result := range results {
		status := "Success"
		rtt := result.RTT.String()
		if result.Error != nil {
			status = "Failed: " + result.Error.Error()
			rtt = "N/A"
		}

		m.tableModel.AddRow([]string{
			fmt.Sprintf("%d", result.Sequence),
			result.Host.Hostname,
			result.Host.IPAddress.String(),
			rtt,
			fmt.Sprintf("%d", result.TTL),
			status,
		})
	}
}

// updateDNSTable updates table model for DNS results
func (m *ResultViewModel) updateDNSTable(result domain.DNSResult) {
	headers := []string{"Name", "Type", "Value", "TTL"}
	m.tableModel = NewTableModel(headers)

	for _, record := range result.Records {
		m.tableModel.AddRow([]string{
			record.Name,
			m.getDNSRecordTypeString(record.Type),
			record.Value,
			fmt.Sprintf("%d", record.TTL),
		})
	}
}

// updateTracerouteTable updates table model for traceroute results
func (m *ResultViewModel) updateTracerouteTable(results []domain.TraceHop) {
	headers := []string{"Hop", "Hostname", "IP Address", "RTT 1", "RTT 2", "RTT 3", "Status"}
	m.tableModel = NewTableModel(headers)

	for _, hop := range results {
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
		
		ipAddr := "-"
		if hop.Host.IPAddress != nil {
			ipAddr = hop.Host.IPAddress.String()
		}
		
		status := "✓ OK"
		if hop.Timeout {
			status = "✗ Timeout"
		}
		
		m.tableModel.AddRow([]string{
			fmt.Sprintf("%d", hop.Number),
			hostname,
			ipAddr,
			rtt1,
			rtt2,
			rtt3,
			status,
		})
	}
}

// updateGenericTable creates a generic table for unknown result types
func (m *ResultViewModel) updateGenericTable() {
	headers := []string{"Property", "Value"}
	m.tableModel = NewTableModel(headers)

	// Add metadata
	for key, value := range m.result.Metadata() {
		m.tableModel.AddRow([]string{key, fmt.Sprintf("%v", value)})
	}
}

// formatPingStatistics formats ping statistics for display
func (m *ResultViewModel) formatPingStatistics(stats interface{}) [][]string {
	// Try to extract statistics from the interface
	statsMap, ok := stats.(map[string]interface{})
	if !ok {
		return [][]string{{"Statistics", "Not available"}}
	}

	var result [][]string

	if sent, ok := statsMap["packets_sent"].(int); ok {
		result = append(result, []string{"Packets Sent", fmt.Sprintf("%d", sent)})
	}

	if received, ok := statsMap["packets_received"].(int); ok {
		result = append(result, []string{"Packets Received", fmt.Sprintf("%d", received)})
	}

	if loss, ok := statsMap["packet_loss_percent"].(float64); ok {
		result = append(result, []string{"Packet Loss", fmt.Sprintf("%.1f%%", loss)})
	}

	if minRTT, ok := statsMap["min_rtt"].(time.Duration); ok {
		result = append(result, []string{"Min RTT", minRTT.String()})
	}

	if maxRTT, ok := statsMap["max_rtt"].(time.Duration); ok {
		result = append(result, []string{"Max RTT", maxRTT.String()})
	}

	if avgRTT, ok := statsMap["avg_rtt"].(time.Duration); ok {
		result = append(result, []string{"Avg RTT", avgRTT.String()})
	}

	if len(result) == 0 {
		result = append(result, []string{"Statistics", "Available but format not recognized"})
	}

	return result
}

// formatTracerouteStatistics formats traceroute statistics for display
func (m *ResultViewModel) formatTracerouteStatistics(stats interface{}) [][]string {
	// Try to extract statistics from the interface
	statsMap, ok := stats.(map[string]interface{})
	if !ok {
		return [][]string{{"Statistics", "Not available"}}
	}

	var result [][]string

	if totalHops, ok := statsMap["total_hops"].(int); ok {
		result = append(result, []string{"Total Hops", fmt.Sprintf("%d", totalHops)})
	}

	if completedHops, ok := statsMap["completed_hops"].(int); ok {
		result = append(result, []string{"Completed Hops", fmt.Sprintf("%d", completedHops)})
	}

	if timeoutHops, ok := statsMap["timeout_hops"].(int); ok {
		result = append(result, []string{"Timeout Hops", fmt.Sprintf("%d", timeoutHops)})
	}

	if successRate, ok := statsMap["success_rate"].(float64); ok {
		result = append(result, []string{"Success Rate", fmt.Sprintf("%.1f%%", successRate)})
	}

	if minRTT, ok := statsMap["min_rtt"].(time.Duration); ok {
		result = append(result, []string{"Min RTT", minRTT.String()})
	}

	if maxRTT, ok := statsMap["max_rtt"].(time.Duration); ok {
		result = append(result, []string{"Max RTT", maxRTT.String()})
	}

	if avgRTT, ok := statsMap["avg_rtt"].(time.Duration); ok {
		result = append(result, []string{"Avg RTT", avgRTT.String()})
	}

	if totalTime, ok := statsMap["total_time"].(time.Duration); ok {
		result = append(result, []string{"Total Time", totalTime.String()})
	}

	if reachedTarget, ok := statsMap["reached_target"].(bool); ok {
		result = append(result, []string{"Reached Target", fmt.Sprintf("%t", reachedTarget)})
	}

	if len(result) == 0 {
		result = append(result, []string{"Statistics", "Available but format not recognized"})
	}

	return result
}

// SetSize implements domain.TUIComponent
func (m *ResultViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	if m.tableModel != nil {
		m.tableModel.SetSize(width, height)
	}

	if m.scrollPager != nil {
		// Reserve space for mode indicator and help text
		pagerHeight := height - 6 // Mode indicator (2) + help text (2) + margins (2)
		if pagerHeight < 1 {
			pagerHeight = 1
		}
		m.scrollPager.SetSize(width, pagerHeight)
	}
}

// SetTheme implements domain.TUIComponent
func (m *ResultViewModel) SetTheme(theme domain.Theme) {
	m.theme = theme

	if m.tableModel != nil {
		m.tableModel.SetTheme(theme)
	}

	if m.scrollPager != nil {
		m.scrollPager.SetTheme(theme)
	}
}

// getDNSRecordTypeString returns a human-readable string for a DNS record type
func (m *ResultViewModel) getDNSRecordTypeString(recordType domain.DNSRecordType) string {
	switch recordType {
	case domain.DNSRecordTypeA:
		return "A"
	case domain.DNSRecordTypeAAAA:
		return "AAAA"
	case domain.DNSRecordTypeMX:
		return "MX"
	case domain.DNSRecordTypeTXT:
		return "TXT"
	case domain.DNSRecordTypeCNAME:
		return "CNAME"
	case domain.DNSRecordTypeNS:
		return "NS"
	case domain.DNSRecordTypeSOA:
		return "SOA"
	case domain.DNSRecordTypePTR:
		return "PTR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", recordType)
	}
}

// Focus implements domain.TUIComponent
func (m *ResultViewModel) Focus() {
	m.focused = true

	if m.mode == ResultViewModeTable && m.tableModel != nil {
		m.tableModel.Focus()
	} else if m.scrollPager != nil {
		m.scrollPager.Focus()
	}
}

// Blur implements domain.TUIComponent
func (m *ResultViewModel) Blur() {
	m.focused = false

	if m.tableModel != nil {
		m.tableModel.Blur()
	}

	if m.scrollPager != nil {
		m.scrollPager.Blur()
	}
}

