// Package tui contains tests for TUI models
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// MockPluginRegistry is a mock implementation of domain.PluginRegistry
type MockPluginRegistry struct {
	mock.Mock
}

func (m *MockPluginRegistry) Register(tool domain.DiagnosticTool) error {
	args := m.Called(tool)
	return args.Error(0)
}

func (m *MockPluginRegistry) Get(name string) (domain.DiagnosticTool, bool) {
	args := m.Called(name)
	return args.Get(0).(domain.DiagnosticTool), args.Bool(1)
}

func (m *MockPluginRegistry) List() []domain.DiagnosticTool {
	args := m.Called()
	return args.Get(0).([]domain.DiagnosticTool)
}

func (m *MockPluginRegistry) Unregister(name string) error {
	args := m.Called(name)
	return args.Error(0)
}



// MockTheme is a mock implementation of domain.Theme
type MockTheme struct {
	mock.Mock
}

func (m *MockTheme) GetColor(element string) string {
	args := m.Called(element)
	return args.String(0)
}

func (m *MockTheme) GetStyle(element string) map[string]interface{} {
	args := m.Called(element)
	return args.Get(0).(map[string]interface{})
}

func (m *MockTheme) SetColor(element, color string) {
	m.Called(element, color)
}

func TestNewMainModel(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)

	assert.NotNil(t, model)
	assert.Equal(t, StateMainMenu, model.state)
	assert.NotNil(t, model.navigation)
	assert.Equal(t, mockRegistry, model.plugins)
	assert.Equal(t, config, model.config)
	assert.Equal(t, mockTheme, model.theme)
	assert.False(t, model.quitting)
}

func TestMainModel_Init(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	cmd := model.Init()

	assert.NotNil(t, cmd)
}

func TestMainModel_Update_WindowSize(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	
	// Test window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, cmd := model.Update(msg)

	mainModel := updatedModel.(*MainModel)
	assert.Equal(t, 100, mainModel.width)
	assert.Equal(t, 50, mainModel.height)
	assert.Nil(t, cmd)
}

func TestMainModel_Update_QuitKey(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	
	// Test quit key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := model.Update(msg)

	mainModel := updatedModel.(*MainModel)
	assert.True(t, mainModel.quitting)
	assert.NotNil(t, cmd)
}

func TestMainModel_Update_BackKey(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	model.state = StateDiagnostic // Set to non-main menu state
	
	// Test back key
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := model.Update(msg)

	mainModel := updatedModel.(*MainModel)
	assert.Equal(t, StateMainMenu, mainModel.state)
	assert.Nil(t, cmd)
}

func TestMainModel_Update_NavigationMsg(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	// Create a mock diagnostic tool
	mockTool := &MockDiagnosticTool{}
	mockTool.On("Name").Return("whois")
	mockTool.On("Description").Return("WHOIS lookup tool")

	// Set up mock expectation for plugin registry
	mockRegistry.On("Get", "whois").Return(mockTool, true)

	model := NewMainModel(mockRegistry, config, mockTheme)
	
	// Test navigation message
	navItem := NavigationItem{ID: "whois", Title: "WHOIS"}
	msg := NavigationMsg{
		Action: NavigationActionSelect,
		Data:   navItem,
	}
	
	updatedModel, cmd := model.Update(msg)

	mainModel := updatedModel.(*MainModel)
	assert.Equal(t, StateDiagnostic, mainModel.state)
	assert.Nil(t, cmd)

	mockRegistry.AssertExpectations(t)
	mockTool.AssertExpectations(t)
}

func TestMainModel_View(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	model.width = 100
	model.height = 50

	view := model.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "NetTraceX")
}

func TestMainModel_View_Quitting(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	model.quitting = true

	view := model.View()
	assert.Equal(t, "Goodbye!\n", view)
}

func TestMainModel_View_NoSize(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	// Don't set width/height

	view := model.View()
	assert.Equal(t, "Loading...", view)
}

func TestMainModel_SetSize(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	model.SetSize(120, 60)

	assert.Equal(t, 120, model.width)
	assert.Equal(t, 60, model.height)
}

func TestMainModel_SetTheme(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	
	newTheme := &MockTheme{}
	model.SetTheme(newTheme)

	assert.Equal(t, newTheme, model.theme)
}

func TestMainModel_Focus_Blur(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	
	// These should not panic
	model.Focus()
	model.Blur()
}

func TestMainModel_HandleBack_FromMainMenu(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	model.state = StateMainMenu
	
	updatedModel, cmd := model.handleBack()
	
	assert.True(t, updatedModel.quitting)
	assert.NotNil(t, cmd)
}

func TestMainModel_HandleBack_FromOtherState(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	model := NewMainModel(mockRegistry, config, mockTheme)
	model.state = StateDiagnostic
	
	updatedModel, cmd := model.handleBack()
	
	assert.Equal(t, StateMainMenu, updatedModel.state)
	assert.Equal(t, model.navigation, updatedModel.activeView)
	assert.Nil(t, cmd)
}

func TestMainModel_SelectNavigationItem(t *testing.T) {
	mockRegistry := &MockPluginRegistry{}
	mockTheme := &MockTheme{}
	config := &domain.Config{}

	// Create mock diagnostic tools for each tool type
	diagnosticTools := []string{"whois", "ping", "traceroute", "dns", "ssl"}
	for _, toolName := range diagnosticTools {
		mockTool := &MockDiagnosticTool{}
		mockTool.On("Name").Return(toolName)
		mockTool.On("Description").Return(toolName + " diagnostic tool")
		mockRegistry.On("Get", toolName).Return(mockTool, true)
	}

	model := NewMainModel(mockRegistry, config, mockTheme)
	
	testCases := []struct {
		itemID       string
		expectedState AppState
	}{
		{"whois", StateDiagnostic},
		{"ping", StateDiagnostic},
		{"traceroute", StateDiagnostic},
		{"dns", StateDiagnostic},
		{"ssl", StateDiagnostic},
		{"settings", StateSettings},
		{"unknown", StateMainMenu}, // Should remain in main menu for unknown items
	}

	for _, tc := range testCases {
		t.Run(tc.itemID, func(t *testing.T) {
			model.state = StateMainMenu // Reset state
			item := NavigationItem{ID: tc.itemID}
			
			updatedModel, _ := model.selectNavigationItem(item)
			
			if tc.itemID == "unknown" {
				assert.Equal(t, StateMainMenu, updatedModel.state)
			} else {
				assert.Equal(t, tc.expectedState, updatedModel.state)
			}
		})
	}

	mockRegistry.AssertExpectations(t)
}

func TestDefaultKeyMap(t *testing.T) {
	keyMap := DefaultKeyMap()
	
	assert.NotNil(t, keyMap.Up)
	assert.NotNil(t, keyMap.Down)
	assert.NotNil(t, keyMap.Left)
	assert.NotNil(t, keyMap.Right)
	assert.NotNil(t, keyMap.Enter)
	assert.NotNil(t, keyMap.Back)
	assert.NotNil(t, keyMap.Quit)
	assert.NotNil(t, keyMap.Help)
	assert.NotNil(t, keyMap.Tab)
}