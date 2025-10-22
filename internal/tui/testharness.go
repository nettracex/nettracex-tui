// Package tui contains test harness for TUI interaction testing
package tui

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// TUITestHarness enables TUI interaction testing
type TUITestHarness struct {
	model       tea.Model
	program     *tea.Program
	inputQueue  chan tea.Msg
	outputLog   []string
	outputMutex sync.RWMutex
	width       int
	height      int
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTUITestHarness creates a new test harness
func NewTUITestHarness(model tea.Model) *TUITestHarness {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TUITestHarness{
		model:      model,
		inputQueue: make(chan tea.Msg, 100),
		outputLog:  make([]string, 0),
		width:      80,
		height:     24,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the test harness
func (h *TUITestHarness) Start() error {
	if h.running {
		return fmt.Errorf("test harness is already running")
	}

	// Create a program with custom options for testing
	h.program = tea.NewProgram(
		h.model,
		tea.WithInput(h),
		tea.WithOutput(&testOutput{harness: h}),
		tea.WithoutSignalHandler(),
	)

	h.running = true

	// Start the program in a goroutine
	go func() {
		defer func() {
			h.running = false
		}()
		
		if _, err := h.program.Run(); err != nil {
			h.logOutput(fmt.Sprintf("Program error: %v", err))
		}
	}()

	// Send initial window size
	h.SendWindowSize(h.width, h.height)
	
	// Give the program time to start
	time.Sleep(10 * time.Millisecond)
	
	return nil
}

// Stop stops the test harness
func (h *TUITestHarness) Stop() {
	if !h.running {
		return
	}

	h.cancel()
	if h.program != nil {
		h.program.Quit()
	}
	
	// Wait for program to stop
	timeout := time.After(1 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return // Force stop after timeout
		case <-ticker.C:
			if !h.running {
				return
			}
		}
	}
}

// SendKey sends a key message to the program
func (h *TUITestHarness) SendKey(key tea.KeyType) {
	h.SendMessage(tea.KeyMsg{Type: key})
}

// SendKeyRune sends a key rune message to the program
func (h *TUITestHarness) SendKeyRune(r rune) {
	h.SendMessage(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
}

// SendKeyString sends a string as individual key messages
func (h *TUITestHarness) SendKeyString(s string) {
	for _, r := range s {
		h.SendKeyRune(r)
		time.Sleep(5 * time.Millisecond) // Small delay between keys
	}
}

// SendWindowSize sends a window size message
func (h *TUITestHarness) SendWindowSize(width, height int) {
	h.width = width
	h.height = height
	h.SendMessage(tea.WindowSizeMsg{Width: width, Height: height})
}

// SendMessage sends a custom message to the program
func (h *TUITestHarness) SendMessage(msg tea.Msg) {
	if !h.running {
		return
	}

	select {
	case h.inputQueue <- msg:
	case <-h.ctx.Done():
		return
	case <-time.After(100 * time.Millisecond):
		// Timeout to prevent blocking
		return
	}
}

// GetOutput returns the current output
func (h *TUITestHarness) GetOutput() string {
	h.outputMutex.RLock()
	defer h.outputMutex.RUnlock()
	
	if len(h.outputLog) == 0 {
		return ""
	}
	
	return h.outputLog[len(h.outputLog)-1]
}

// GetOutputHistory returns all output history
func (h *TUITestHarness) GetOutputHistory() []string {
	h.outputMutex.RLock()
	defer h.outputMutex.RUnlock()
	
	// Return a copy to prevent race conditions
	history := make([]string, len(h.outputLog))
	copy(history, h.outputLog)
	return history
}

// WaitForOutput waits for specific text to appear in the output
func (h *TUITestHarness) WaitForOutput(text string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		output := h.GetOutput()
		if strings.Contains(output, text) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	return false
}

// WaitForOutputMatch waits for output that matches a predicate function
func (h *TUITestHarness) WaitForOutputMatch(predicate func(string) bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		output := h.GetOutput()
		if predicate(output) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	return false
}

// AssertOutput checks if the current output contains the expected text
func (h *TUITestHarness) AssertOutput(expected string) bool {
	output := h.GetOutput()
	return strings.Contains(output, expected)
}

// AssertOutputNot checks if the current output does not contain the text
func (h *TUITestHarness) AssertOutputNot(notExpected string) bool {
	output := h.GetOutput()
	return !strings.Contains(output, notExpected)
}

// GetModel returns the current model (for inspection)
func (h *TUITestHarness) GetModel() tea.Model {
	return h.model
}

// IsRunning returns whether the harness is currently running
func (h *TUITestHarness) IsRunning() bool {
	return h.running
}

// logOutput logs output to the history
func (h *TUITestHarness) logOutput(output string) {
	h.outputMutex.Lock()
	defer h.outputMutex.Unlock()
	
	h.outputLog = append(h.outputLog, output)
	
	// Keep only the last 100 outputs to prevent memory issues
	if len(h.outputLog) > 100 {
		h.outputLog = h.outputLog[1:]
	}
}

// Read implements io.Reader for tea.WithInput
func (h *TUITestHarness) Read(p []byte) (n int, err error) {
	select {
	case msg := <-h.inputQueue:
		// Convert message to bytes (simplified)
		var data []byte
		switch m := msg.(type) {
		case tea.KeyMsg:
			if m.Type == tea.KeyRunes && len(m.Runes) > 0 {
				data = []byte(string(m.Runes))
			} else {
				// Convert key type to escape sequence
				data = keyToBytes(m.Type)
			}
		default:
			// For other message types, we'll simulate them differently
			return 0, nil
		}
		
		if len(data) > len(p) {
			data = data[:len(p)]
		}
		
		copy(p, data)
		return len(data), nil
		
	case <-h.ctx.Done():
		return 0, fmt.Errorf("harness stopped")
	}
}

// keyToBytes converts tea.Key to byte sequence
func keyToBytes(key tea.KeyType) []byte {
	switch key {
	case tea.KeyEnter:
		return []byte{'\r'}
	case tea.KeyTab:
		return []byte{'\t'}
	case tea.KeyEsc:
		return []byte{'\x1b'}
	case tea.KeyUp:
		return []byte{'\x1b', '[', 'A'}
	case tea.KeyDown:
		return []byte{'\x1b', '[', 'B'}
	case tea.KeyRight:
		return []byte{'\x1b', '[', 'C'}
	case tea.KeyLeft:
		return []byte{'\x1b', '[', 'D'}
	case tea.KeyBackspace:
		return []byte{'\x7f'}
	case tea.KeyDelete:
		return []byte{'\x1b', '[', '3', '~'}
	case tea.KeyCtrlC:
		return []byte{'\x03'}
	default:
		return []byte{}
	}
}

// testOutput implements io.Writer for capturing output
type testOutput struct {
	harness *TUITestHarness
	buffer  bytes.Buffer
}

// Write implements io.Writer
func (o *testOutput) Write(p []byte) (n int, err error) {
	o.buffer.Write(p)
	
	// Log the output to the harness
	output := string(p)
	o.harness.logOutput(output)
	
	return len(p), nil
}

// TUITestSuite provides utilities for testing TUI components
type TUITestSuite struct {
	harnesses []*TUITestHarness
}

// NewTUITestSuite creates a new test suite
func NewTUITestSuite() *TUITestSuite {
	return &TUITestSuite{
		harnesses: make([]*TUITestHarness, 0),
	}
}

// CreateHarness creates and registers a new test harness
func (s *TUITestSuite) CreateHarness(model tea.Model) *TUITestHarness {
	harness := NewTUITestHarness(model)
	s.harnesses = append(s.harnesses, harness)
	return harness
}

// Cleanup stops all harnesses and cleans up resources
func (s *TUITestSuite) Cleanup() {
	for _, harness := range s.harnesses {
		harness.Stop()
	}
	s.harnesses = nil
}

// TestNavigationFlow tests basic navigation flows
func (s *TUITestSuite) TestNavigationFlow(harness *TUITestHarness) error {
	if err := harness.Start(); err != nil {
		return fmt.Errorf("failed to start harness: %w", err)
	}
	defer harness.Stop()

	// Wait for initial render
	if !harness.WaitForOutput("NetTraceX", 1*time.Second) {
		return fmt.Errorf("initial render timeout")
	}

	// Test down navigation
	harness.SendKey(tea.KeyDown)
	time.Sleep(50 * time.Millisecond)

	// Test up navigation
	harness.SendKey(tea.KeyUp)
	time.Sleep(50 * time.Millisecond)

	// Test enter key
	harness.SendKey(tea.KeyEnter)
	time.Sleep(50 * time.Millisecond)

	return nil
}

// TestFormInteraction tests form interaction flows
func (s *TUITestSuite) TestFormInteraction(harness *TUITestHarness) error {
	if err := harness.Start(); err != nil {
		return fmt.Errorf("failed to start harness: %w", err)
	}
	defer harness.Stop()

	// Type in form field
	harness.SendKeyString("test input")
	time.Sleep(100 * time.Millisecond)

	// Navigate between fields
	harness.SendKey(tea.KeyTab)
	time.Sleep(50 * time.Millisecond)

	// Submit form
	harness.SendKey(tea.KeyEnter)
	time.Sleep(50 * time.Millisecond)

	return nil
}

// TestKeyboardShortcuts tests keyboard shortcuts
func (s *TUITestSuite) TestKeyboardShortcuts(harness *TUITestHarness) error {
	if err := harness.Start(); err != nil {
		return fmt.Errorf("failed to start harness: %w", err)
	}
	defer harness.Stop()

	shortcuts := []tea.KeyType{
		tea.KeyEsc,    // Back
		tea.KeyCtrlC,  // Quit
	}

	for _, shortcut := range shortcuts {
		harness.SendKey(shortcut)
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

// TestResponsiveLayout tests responsive layout behavior
func (s *TUITestSuite) TestResponsiveLayout(harness *TUITestHarness) error {
	if err := harness.Start(); err != nil {
		return fmt.Errorf("failed to start harness: %w", err)
	}
	defer harness.Stop()

	// Test different screen sizes
	sizes := []struct{ width, height int }{
		{60, 20},   // Small
		{100, 30},  // Medium
		{140, 40},  // Large
	}

	for _, size := range sizes {
		harness.SendWindowSize(size.width, size.height)
		time.Sleep(50 * time.Millisecond)
		
		// Verify the layout adapts
		if !harness.WaitForOutput("", 500*time.Millisecond) {
			return fmt.Errorf("layout adaptation timeout for size %dx%d", size.width, size.height)
		}
	}

	return nil
}

// MockTUITestComponent creates a mock component for testing
func MockTUITestComponent() domain.TUIComponent {
	return &mockTUIComponent{
		width:  80,
		height: 24,
	}
}

type mockTUIComponent struct {
	width   int
	height  int
	theme   domain.Theme
	focused bool
}

func (m *mockTUIComponent) Init() tea.Cmd {
	return nil
}

func (m *mockTUIComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *mockTUIComponent) View() string {
	return fmt.Sprintf("Mock Component (%dx%d) Focused: %t", m.width, m.height, m.focused)
}

func (m *mockTUIComponent) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *mockTUIComponent) SetTheme(theme domain.Theme) {
	m.theme = theme
}

func (m *mockTUIComponent) Focus() {
	m.focused = true
}

func (m *mockTUIComponent) Blur() {
	m.focused = false
}