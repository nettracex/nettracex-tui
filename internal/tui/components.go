// Package tui contains reusable TUI components
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// FormField represents a single form field
type FormField struct {
	Key         string
	Label       string
	Input       textinput.Model
	Required    bool
	Validator   domain.Validator
	HelpText    string
	ErrorText   string
}

// FormModel provides input forms with validation
type FormModel struct {
	fields    []FormField
	focused   int
	width     int
	height    int
	theme     domain.Theme
	title     string
	validator domain.Validator
	submitted bool
	keyMap    KeyMap
}

// NewFormModel creates a new form model
func NewFormModel(title string) *FormModel {
	return &FormModel{
		title:   title,
		focused: 0,
		keyMap:  DefaultKeyMap(),
	}
}

// AddField adds a field to the form
func (m *FormModel) AddField(key, label string, required bool) {
	input := textinput.New()
	input.Placeholder = label
	input.CharLimit = 256
	
	field := FormField{
		Key:      key,
		Label:    label,
		Input:    input,
		Required: required,
	}
	
	m.fields = append(m.fields, field)
	
	// Focus the first field
	if len(m.fields) == 1 {
		field.Input.Focus()
	}
}

// SetFieldValue sets the value of a field
func (m *FormModel) SetFieldValue(key, value string) {
	for i := range m.fields {
		if m.fields[i].Key == key {
			m.fields[i].Input.SetValue(value)
			break
		}
	}
}

// GetFieldValue gets the value of a field
func (m *FormModel) GetFieldValue(key string) string {
	for _, field := range m.fields {
		if field.Key == key {
			return field.Input.Value()
		}
	}
	return ""
}

// GetValues returns all form values as a map
func (m *FormModel) GetValues() map[string]string {
	values := make(map[string]string)
	for _, field := range m.fields {
		values[field.Key] = field.Input.Value()
	}
	return values
}

// Init implements tea.Model
func (m *FormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (m *FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Tab):
			m.nextField()
			return m, nil

		case key.Matches(msg, m.keyMap.Up):
			m.prevField()
			return m, nil

		case key.Matches(msg, m.keyMap.Down):
			m.nextField()
			return m, nil

		case key.Matches(msg, m.keyMap.Enter):
			if m.validate() {
				m.submitted = true
				return m, func() tea.Msg {
					return FormSubmitMsg{Values: m.GetValues()}
				}
			}
			return m, nil
		}
	}

	// Update the focused field
	if m.focused >= 0 && m.focused < len(m.fields) {
		m.fields[m.focused].Input, cmd = m.fields[m.focused].Input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *FormModel) View() string {
	var content []string

	// Title
	if m.title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(1, 0)
		content = append(content, titleStyle.Render(m.title))
		content = append(content, "")
	}

	// Form fields
	for i, field := range m.fields {
		fieldContent := m.renderField(field, i == m.focused)
		content = append(content, fieldContent)
		content = append(content, "") // Add spacing
	}

	// Instructions
	if !m.submitted {
		instructionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
		
		instructions := "Tab/↑↓: navigate • Enter: submit • Esc: back"
		content = append(content, instructionStyle.Render(instructions))
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// renderField renders a single form field
func (m *FormModel) renderField(field FormField, focused bool) string {
	var parts []string

	// Label
	labelStyle := lipgloss.NewStyle().Bold(true)
	if field.Required {
		labelStyle = labelStyle.Foreground(lipgloss.Color("196"))
		parts = append(parts, labelStyle.Render(field.Label+" *"))
	} else {
		parts = append(parts, labelStyle.Render(field.Label))
	}

	// Input
	inputStyle := lipgloss.NewStyle().
		Width(m.width - 4).
		Padding(0, 1)
	
	if focused {
		inputStyle = inputStyle.Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	} else {
		inputStyle = inputStyle.Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	}

	parts = append(parts, inputStyle.Render(field.Input.View()))

	// Error text
	if field.ErrorText != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Italic(true)
		parts = append(parts, errorStyle.Render("Error: "+field.ErrorText))
	}

	// Help text
	if field.HelpText != "" && field.ErrorText == "" {
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
		parts = append(parts, helpStyle.Render(field.HelpText))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// nextField moves focus to the next field
func (m *FormModel) nextField() {
	if len(m.fields) == 0 {
		return
	}

	// Blur current field
	if m.focused >= 0 && m.focused < len(m.fields) {
		m.fields[m.focused].Input.Blur()
	}

	// Move to next field
	m.focused++
	if m.focused >= len(m.fields) {
		m.focused = 0
	}

	// Focus new field
	m.fields[m.focused].Input.Focus()
}

// prevField moves focus to the previous field
func (m *FormModel) prevField() {
	if len(m.fields) == 0 {
		return
	}

	// Blur current field
	if m.focused >= 0 && m.focused < len(m.fields) {
		m.fields[m.focused].Input.Blur()
	}

	// Move to previous field
	m.focused--
	if m.focused < 0 {
		m.focused = len(m.fields) - 1
	}

	// Focus new field
	m.fields[m.focused].Input.Focus()
}

// validate validates all form fields
func (m *FormModel) validate() bool {
	valid := true

	for i := range m.fields {
		field := &m.fields[i]
		field.ErrorText = ""

		// Check required fields
		if field.Required && strings.TrimSpace(field.Input.Value()) == "" {
			field.ErrorText = "This field is required"
			valid = false
			continue
		}

		// Run custom validator if present
		if field.Validator != nil {
			if err := field.Validator.Validate(field.Input.Value()); err != nil {
				field.ErrorText = err.Error()
				valid = false
			}
		}
	}

	return valid
}

// SetSize implements domain.TUIComponent
func (m *FormModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTheme implements domain.TUIComponent
func (m *FormModel) SetTheme(theme domain.Theme) {
	m.theme = theme
}

// Focus implements domain.TUIComponent
func (m *FormModel) Focus() {
	if len(m.fields) > 0 && m.focused >= 0 && m.focused < len(m.fields) {
		m.fields[m.focused].Input.Focus()
	}
}

// Blur implements domain.TUIComponent
func (m *FormModel) Blur() {
	if len(m.fields) > 0 && m.focused >= 0 && m.focused < len(m.fields) {
		m.fields[m.focused].Input.Blur()
	}
}

// FormSubmitMsg represents a form submission message
type FormSubmitMsg struct {
	Values map[string]string
}

// TableModel displays tabular data with sorting and filtering
type TableModel struct {
	headers   []string
	rows      [][]string
	sortBy    int
	sortDesc  bool
	filter    string
	selected  int
	width     int
	height    int
	theme     domain.Theme
	focused   bool
	keyMap    KeyMap
}

// NewTableModel creates a new table model
func NewTableModel(headers []string) *TableModel {
	return &TableModel{
		headers:  headers,
		rows:     [][]string{},
		sortBy:   -1,
		selected: 0,
		focused:  true,
		keyMap:   DefaultKeyMap(),
	}
}

// SetData sets the table data
func (m *TableModel) SetData(rows [][]string) {
	m.rows = rows
	if m.selected >= len(m.rows) {
		m.selected = len(m.rows) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

// AddRow adds a row to the table
func (m *TableModel) AddRow(row []string) {
	m.rows = append(m.rows, row)
}

// Init implements tea.Model
func (m *TableModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keyMap.Up):
			m.selected--
			if m.selected < 0 {
				m.selected = len(m.rows) - 1
			}

		case key.Matches(msg, m.keyMap.Down):
			m.selected++
			if m.selected >= len(m.rows) {
				m.selected = 0
			}

		case key.Matches(msg, m.keyMap.Enter):
			if m.selected >= 0 && m.selected < len(m.rows) {
				return m, func() tea.Msg {
					return TableSelectMsg{
						Row:   m.selected,
						Data:  m.rows[m.selected],
					}
				}
			}
		}
	}

	return m, nil
}

// View implements tea.Model
func (m *TableModel) View() string {
	if len(m.headers) == 0 {
		return "No table headers defined"
	}

	var content []string

	// Calculate column widths
	colWidths := m.calculateColumnWidths()

	// Render headers
	headerRow := m.renderHeaderRow(colWidths)
	content = append(content, headerRow)

	// Render separator
	separator := m.renderSeparator(colWidths)
	content = append(content, separator)

	// Render data rows
	filteredRows := m.getFilteredRows()
	for i, row := range filteredRows {
		rowContent := m.renderDataRow(row, i == m.selected, colWidths)
		content = append(content, rowContent)
	}

	if len(filteredRows) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true).
			Padding(1, 0)
		content = append(content, emptyStyle.Render("No data available"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// calculateColumnWidths calculates optimal column widths
func (m *TableModel) calculateColumnWidths() []int {
	if m.width == 0 {
		// Default widths if no size set
		widths := make([]int, len(m.headers))
		for i := range widths {
			widths[i] = 15
		}
		return widths
	}

	// Calculate available width (accounting for borders and padding)
	availableWidth := m.width - (len(m.headers) * 3) - 2

	// Start with header widths
	widths := make([]int, len(m.headers))
	for i, header := range m.headers {
		widths[i] = len(header)
	}

	// Check data rows for maximum width
	for _, row := range m.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Distribute available width proportionally
	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}

	if totalWidth > availableWidth {
		// Scale down proportionally
		for i := range widths {
			widths[i] = (widths[i] * availableWidth) / totalWidth
			if widths[i] < 5 {
				widths[i] = 5 // Minimum width
			}
		}
	}

	return widths
}

// renderHeaderRow renders the table header
func (m *TableModel) renderHeaderRow(colWidths []int) string {
	var cells []string

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 1)

	for i, header := range m.headers {
		width := colWidths[i]
		cell := headerStyle.Width(width).Render(header)
		cells = append(cells, cell)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// renderSeparator renders a separator line
func (m *TableModel) renderSeparator(colWidths []int) string {
	var parts []string
	for _, width := range colWidths {
		parts = append(parts, strings.Repeat("─", width+2))
	}
	return strings.Join(parts, "┼")
}

// renderDataRow renders a single data row
func (m *TableModel) renderDataRow(row []string, selected bool, colWidths []int) string {
	var cells []string

	style := lipgloss.NewStyle().Padding(0, 1)
	if selected {
		style = style.
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))
	}

	for i, cell := range row {
		if i >= len(colWidths) {
			break
		}
		width := colWidths[i]
		
		// Truncate cell if too long
		if len(cell) > width {
			cell = cell[:width-3] + "..."
		}
		
		styledCell := style.Width(width).Render(cell)
		cells = append(cells, styledCell)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// getFilteredRows returns rows filtered by the current filter
func (m *TableModel) getFilteredRows() [][]string {
	if m.filter == "" {
		return m.rows
	}

	var filtered [][]string
	filterLower := strings.ToLower(m.filter)

	for _, row := range m.rows {
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), filterLower) {
				filtered = append(filtered, row)
				break
			}
		}
	}

	return filtered
}

// SetSize implements domain.TUIComponent
func (m *TableModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTheme implements domain.TUIComponent
func (m *TableModel) SetTheme(theme domain.Theme) {
	m.theme = theme
}

// Focus implements domain.TUIComponent
func (m *TableModel) Focus() {
	m.focused = true
}

// Blur implements domain.TUIComponent
func (m *TableModel) Blur() {
	m.focused = false
}

// SortBy sorts the table by the specified column
func (m *TableModel) SortBy(column int, descending bool) {
	if column < 0 || column >= len(m.headers) {
		return
	}

	m.sortBy = column
	m.sortDesc = descending

	sort.Slice(m.rows, func(i, j int) bool {
		if column >= len(m.rows[i]) || column >= len(m.rows[j]) {
			return false
		}

		a, b := m.rows[i][column], m.rows[j][column]
		if descending {
			return a > b
		}
		return a < b
	})
}

// SetFilter sets the table filter
func (m *TableModel) SetFilter(filter string) {
	m.filter = filter
	m.selected = 0 // Reset selection when filtering
}

// TableSelectMsg represents a table row selection message
type TableSelectMsg struct {
	Row  int
	Data []string
}

// ProgressModel shows operation progress
type ProgressModel struct {
	current   int
	total     int
	message   string
	animated  bool
	width     int
	height    int
	theme     domain.Theme
}

// NewProgressModel creates a new progress model
func NewProgressModel() *ProgressModel {
	return &ProgressModel{
		animated: true,
	}
}

// SetProgress sets the current progress
func (m *ProgressModel) SetProgress(current, total int) {
	m.current = current
	m.total = total
}

// SetMessage sets the progress message
func (m *ProgressModel) SetMessage(message string) {
	m.message = message
}

// Init implements tea.Model
func (m *ProgressModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View implements tea.Model
func (m *ProgressModel) View() string {
	if m.total == 0 {
		return "No progress to display"
	}

	// Calculate progress percentage
	percentage := float64(m.current) / float64(m.total)
	if percentage > 1.0 {
		percentage = 1.0
	}

	// Progress bar width
	barWidth := m.width - 20 // Leave space for percentage and padding
	if barWidth < 10 {
		barWidth = 10
	}

	// Calculate filled width
	filledWidth := int(float64(barWidth) * percentage)

	// Create progress bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", barWidth-filledWidth)
	progressBar := filled + empty

	// Style the progress bar
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Background(lipgloss.Color("240"))

	styledBar := progressStyle.Render(progressBar)

	// Format percentage
	percentText := fmt.Sprintf("%3.0f%%", percentage*100)

	// Format counter
	counterText := fmt.Sprintf("(%d/%d)", m.current, m.total)

	// Combine elements
	var parts []string
	if m.message != "" {
		messageStyle := lipgloss.NewStyle().Bold(true)
		parts = append(parts, messageStyle.Render(m.message))
	}

	progressLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		styledBar,
		" ",
		percentText,
		" ",
		counterText,
	)
	parts = append(parts, progressLine)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// SetSize implements domain.TUIComponent
func (m *ProgressModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTheme implements domain.TUIComponent
func (m *ProgressModel) SetTheme(theme domain.Theme) {
	m.theme = theme
}

// Focus implements domain.TUIComponent
func (m *ProgressModel) Focus() {
	// Progress bars don't need focus
}

// Blur implements domain.TUIComponent
func (m *ProgressModel) Blur() {
	// Progress bars don't need focus
}

// IsComplete returns true if progress is complete
func (m *ProgressModel) IsComplete() bool {
	return m.current >= m.total && m.total > 0
}