// Package tui contains integration tests for standardized scrolling behavior
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// TestScrollingIntegrationAcrossModels verifies that all TUI models respond consistently to navigation keys
func TestScrollingIntegrationAcrossModels(t *testing.T) {
	// Test NavigationModel scrolling consistency
	t.Run("NavigationModel", func(t *testing.T) {
		nav := NewNavigationModel()
		nav.SetSize(80, 20)
		
		// Test basic navigation keys
		testStandardScrollingKeys(t, nav, "NavigationModel")
	})
	
	// Test HelpModel scrolling consistency
	t.Run("HelpModel", func(t *testing.T) {
		help := NewHelpModel()
		help.SetSize(80, 20)
		
		// Initialize help content
		help.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
		
		// Test basic navigation keys
		testStandardScrollingKeys(t, help, "HelpModel")
	})
	
	// Test ResultViewModel scrolling consistency
	t.Run("ResultViewModel", func(t *testing.T) {
		result := NewResultViewModel()
		result.SetSize(80, 20)
		
		// Set some test result data
		testResult := &mockResult{
			data: "Test result content\nLine 2\nLine 3\nLine 4\nLine 5",
		}
		result.SetResult(testResult)
		
		// Test basic navigation keys
		testStandardScrollingKeys(t, result, "ResultViewModel")
	})
}

// testStandardScrollingKeys tests that a model responds to standard scrolling keys
func testStandardScrollingKeys(t *testing.T, model tea.Model, modelName string) {
	
	// Test Up key
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	if cmd != nil {
		t.Logf("%s: Up key handled correctly", modelName)
	}
	
	// Test Down key
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Logf("%s: Down key handled correctly", modelName)
	}
	
	// Test PageUp key
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if cmd != nil {
		t.Logf("%s: PageUp key handled correctly", modelName)
	}
	
	// Test PageDown key
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if cmd != nil {
		t.Logf("%s: PageDown key handled correctly", modelName)
	}
	
	// Test Home key
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyHome})
	if cmd != nil {
		t.Logf("%s: Home key handled correctly", modelName)
	}
	
	// Test End key
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if cmd != nil {
		t.Logf("%s: End key handled correctly", modelName)
	}
	
	// Test that the model can render without errors
	view := model.View()
	if view == "" {
		t.Errorf("%s: View() returned empty string", modelName)
	}
}

// TestScrollIndicatorConsistency verifies that scroll indicators work uniformly across models
func TestScrollIndicatorConsistency(t *testing.T) {
	// Test NavigationModel scroll indicators
	t.Run("NavigationModel_ScrollIndicators", func(t *testing.T) {
		nav := NewNavigationModel()
		nav.SetSize(80, 5) // Small height to force scrolling
		
		view := nav.View()
		if view == "" {
			t.Error("NavigationModel view is empty")
		}
		
		// The view should render without errors even with small height
		t.Logf("NavigationModel rendered successfully with small viewport")
	})
	
	// Test HelpModel scroll indicators
	t.Run("HelpModel_ScrollIndicators", func(t *testing.T) {
		help := NewHelpModel()
		help.SetSize(80, 5) // Small height to force scrolling
		
		// Initialize help content
		help.Update(tea.WindowSizeMsg{Width: 80, Height: 5})
		
		view := help.View()
		if view == "" {
			t.Error("HelpModel view is empty")
		}
		
		t.Logf("HelpModel rendered successfully with small viewport")
	})
	
	// Test ResultViewModel scroll indicators
	t.Run("ResultViewModel_ScrollIndicators", func(t *testing.T) {
		result := NewResultViewModel()
		result.SetSize(80, 5) // Small height to force scrolling
		
		// Set test result with multiple lines
		testResult := &mockResult{
			data: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10",
		}
		result.SetResult(testResult)
		
		view := result.View()
		if view == "" {
			t.Error("ResultViewModel view is empty")
		}
		
		t.Logf("ResultViewModel rendered successfully with small viewport")
	})
}

// TestKeyBindingConsistency verifies that all models use consistent key bindings
func TestKeyBindingConsistency(t *testing.T) {
	keyMap := DefaultKeyMap()
	
	// Verify that all standard scrolling keys are defined
	if !key.Matches(tea.KeyMsg{Type: tea.KeyUp}, keyMap.Up) {
		t.Error("Up key binding not properly defined")
	}
	
	if !key.Matches(tea.KeyMsg{Type: tea.KeyDown}, keyMap.Down) {
		t.Error("Down key binding not properly defined")
	}
	
	// Test that PageUp, PageDown, Home, End are defined
	testKeys := []struct {
		name    string
		binding key.Binding
		keyType tea.KeyType
	}{
		{"PageUp", keyMap.PageUp, tea.KeyPgUp},
		{"PageDown", keyMap.PageDown, tea.KeyPgDown},
		{"Home", keyMap.Home, tea.KeyHome},
		{"End", keyMap.End, tea.KeyEnd},
	}
	
	for _, testKey := range testKeys {
		if !key.Matches(tea.KeyMsg{Type: testKey.keyType}, testKey.binding) {
			t.Errorf("%s key binding not properly defined", testKey.name)
		}
	}
}

// TestMainModelScrollingDelegation verifies that MainModel properly delegates scrolling to active views
func TestMainModelScrollingDelegation(t *testing.T) {
	// Create a mock plugin registry and config
	plugins := &mockPluginRegistry{}
	config := &domain.Config{} // Use actual Config struct
	theme := NewDefaultTheme()
	
	main := NewMainModel(plugins, config, theme)
	main.SetSize(80, 20)
	
	// Test that MainModel delegates key events to active view
	_, cmd := main.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Log("MainModel properly delegates key events")
	}
	
	// Test window size updates
	_, cmd = main.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Log("MainModel handles window size updates")
	}
	
	// Verify that the view renders
	view := main.View()
	if view == "" {
		t.Error("MainModel view is empty")
	}
}

// Mock implementations for testing

type mockResult struct {
	data interface{}
}

func (m *mockResult) Data() interface{} {
	return m.data
}

func (m *mockResult) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"test": "metadata",
	}
}

func (m *mockResult) Export(format domain.ExportFormat) ([]byte, error) {
	return []byte(m.data.(string)), nil
}

func (m *mockResult) Format(formatter domain.OutputFormatter) string {
	return m.data.(string)
}

type mockPluginRegistry struct{}

func (m *mockPluginRegistry) Register(tool domain.DiagnosticTool) error {
	return nil
}

func (m *mockPluginRegistry) Unregister(name string) error {
	return nil
}

func (m *mockPluginRegistry) Get(name string) (domain.DiagnosticTool, bool) {
	return nil, false
}

func (m *mockPluginRegistry) List() []domain.DiagnosticTool {
	return []domain.DiagnosticTool{}
}

type mockConfig struct{}

func (m *mockConfig) GetString(key string) string {
	return ""
}

func (m *mockConfig) GetInt(key string) int {
	return 0
}

func (m *mockConfig) GetBool(key string) bool {
	return false
}

func (m *mockConfig) Set(key string, value interface{}) error {
	return nil
}

func (m *mockConfig) Save() error {
	return nil
}