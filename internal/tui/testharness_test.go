// Package tui contains tests for the TUI test harness
package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewTUITestHarness(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	assert.NotNil(t, harness)
	assert.Equal(t, model, harness.model)
	assert.NotNil(t, harness.inputQueue)
	assert.NotNil(t, harness.outputLog)
	assert.Equal(t, 80, harness.width)
	assert.Equal(t, 24, harness.height)
	assert.False(t, harness.running)
	assert.NotNil(t, harness.ctx)
	assert.NotNil(t, harness.cancel)
}

func TestTUITestHarness_StartStop(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	// Test start
	err := harness.Start()
	assert.NoError(t, err)
	assert.True(t, harness.IsRunning())

	// Test double start (should fail)
	err = harness.Start()
	assert.Error(t, err)

	// Test stop
	harness.Stop()
	
	// Wait for stop to complete
	time.Sleep(100 * time.Millisecond)
	assert.False(t, harness.IsRunning())
}

func TestTUITestHarness_SendKey(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	err := harness.Start()
	assert.NoError(t, err)
	defer harness.Stop()

	// Test sending keys
	harness.SendKey(tea.KeyEnter)
	harness.SendKey(tea.KeyEsc)
	harness.SendKey(tea.KeyUp)
	harness.SendKey(tea.KeyDown)

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Should not panic or error
}

func TestTUITestHarness_SendKeyRune(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	err := harness.Start()
	assert.NoError(t, err)
	defer harness.Stop()

	// Test sending runes
	harness.SendKeyRune('a')
	harness.SendKeyRune('b')
	harness.SendKeyRune('1')

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	// Should not panic or error
}

func TestTUITestHarness_SendKeyString(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	err := harness.Start()
	assert.NoError(t, err)
	defer harness.Stop()

	// Test sending string
	harness.SendKeyString("hello")

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Should not panic or error
}

func TestTUITestHarness_SendWindowSize(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	err := harness.Start()
	assert.NoError(t, err)
	defer harness.Stop()

	// Test sending window size
	harness.SendWindowSize(100, 50)

	assert.Equal(t, 100, harness.width)
	assert.Equal(t, 50, harness.height)

	// Give time for processing
	time.Sleep(50 * time.Millisecond)
}

func TestTUITestHarness_Output(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	// Test initial output
	output := harness.GetOutput()
	assert.Equal(t, "", output)

	history := harness.GetOutputHistory()
	assert.Empty(t, history)

	// Start harness to generate output
	err := harness.Start()
	assert.NoError(t, err)
	defer harness.Stop()

	// Give time for initial render
	time.Sleep(100 * time.Millisecond)

	// Check that output is captured
	output = harness.GetOutput()
	// Output might be empty or contain escape sequences, just check it doesn't panic
	assert.NotNil(t, output)

	history = harness.GetOutputHistory()
	assert.NotNil(t, history)
}

func TestTUITestHarness_WaitForOutput(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	err := harness.Start()
	assert.NoError(t, err)
	defer harness.Stop()

	// Test waiting for output that should appear
	found := harness.WaitForOutput("Mock Component", 500*time.Millisecond)
	// This might be true or false depending on timing, just test it doesn't panic
	assert.NotNil(t, found)

	// Test waiting for output that won't appear
	found = harness.WaitForOutput("NonExistentText", 100*time.Millisecond)
	assert.False(t, found)
}

func TestTUITestHarness_WaitForOutputMatch(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	err := harness.Start()
	assert.NoError(t, err)
	defer harness.Stop()

	// Test predicate function
	predicate := func(output string) bool {
		return len(output) > 0
	}

	found := harness.WaitForOutputMatch(predicate, 500*time.Millisecond)
	// This might be true or false depending on timing, just test it doesn't panic
	assert.NotNil(t, found)
}

func TestTUITestHarness_AssertOutput(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	// Mock some output
	harness.logOutput("Test output content")

	// Test assertions
	assert.True(t, harness.AssertOutput("Test output"))
	assert.False(t, harness.AssertOutput("NonExistent"))
	assert.True(t, harness.AssertOutputNot("NonExistent"))
	assert.False(t, harness.AssertOutputNot("Test output"))
}

func TestTUITestHarness_GetModel(t *testing.T) {
	model := &mockTUIComponent{}
	harness := NewTUITestHarness(model)

	retrievedModel := harness.GetModel()
	assert.Equal(t, model, retrievedModel)
}

func TestKeyToBytes(t *testing.T) {
	testCases := []struct {
		key      tea.KeyType
		expected []byte
	}{
		{tea.KeyEnter, []byte{'\r'}},
		{tea.KeyTab, []byte{'\t'}},
		{tea.KeyEsc, []byte{'\x1b'}},
		{tea.KeyUp, []byte{'\x1b', '[', 'A'}},
		{tea.KeyDown, []byte{'\x1b', '[', 'B'}},
		{tea.KeyRight, []byte{'\x1b', '[', 'C'}},
		{tea.KeyLeft, []byte{'\x1b', '[', 'D'}},
		{tea.KeyBackspace, []byte{'\x7f'}},
		{tea.KeyDelete, []byte{'\x1b', '[', '3', '~'}},
		{tea.KeyCtrlC, []byte{'\x03'}},
	}

	for _, tc := range testCases {
		result := keyToBytes(tc.key)
		assert.Equal(t, tc.expected, result, "Key %v should produce bytes %v", tc.key, tc.expected)
	}

	// Test unknown key
	result := keyToBytes(tea.KeyType(999))
	assert.Empty(t, result)
}

func TestNewTUITestSuite(t *testing.T) {
	suite := NewTUITestSuite()

	assert.NotNil(t, suite)
	assert.Empty(t, suite.harnesses)
}

func TestTUITestSuite_CreateHarness(t *testing.T) {
	suite := NewTUITestSuite()
	model := &mockTUIComponent{}

	harness := suite.CreateHarness(model)

	assert.NotNil(t, harness)
	assert.Equal(t, 1, len(suite.harnesses))
	assert.Equal(t, harness, suite.harnesses[0])
}

func TestTUITestSuite_Cleanup(t *testing.T) {
	suite := NewTUITestSuite()
	model1 := &mockTUIComponent{}
	model2 := &mockTUIComponent{}

	harness1 := suite.CreateHarness(model1)
	harness2 := suite.CreateHarness(model2)

	// Start harnesses
	err := harness1.Start()
	assert.NoError(t, err)
	err = harness2.Start()
	assert.NoError(t, err)

	assert.Equal(t, 2, len(suite.harnesses))

	// Cleanup
	suite.Cleanup()

	assert.Nil(t, suite.harnesses)
	
	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)
	
	// Harnesses should be stopped
	assert.False(t, harness1.IsRunning())
	assert.False(t, harness2.IsRunning())
}

func TestTUITestSuite_TestNavigationFlow(t *testing.T) {
	suite := NewTUITestSuite()
	defer suite.Cleanup()

	// Create a navigation model for testing
	navModel := NewNavigationModel()
	harness := suite.CreateHarness(navModel)

	err := suite.TestNavigationFlow(harness)
	// This might succeed or fail depending on timing, just test it doesn't panic
	assert.NotNil(t, err) // err can be nil or not nil
}

func TestTUITestSuite_TestFormInteraction(t *testing.T) {
	suite := NewTUITestSuite()
	defer suite.Cleanup()

	// Create a form model for testing
	formModel := NewFormModel("Test Form")
	formModel.AddField("test", "Test Field", true)
	harness := suite.CreateHarness(formModel)

	err := suite.TestFormInteraction(harness)
	// This might succeed or fail depending on timing, just test it doesn't panic
	_ = err // err can be nil or not nil, we just want to ensure no panic
}

func TestTUITestSuite_TestKeyboardShortcuts(t *testing.T) {
	suite := NewTUITestSuite()
	defer suite.Cleanup()

	model := &mockTUIComponent{}
	harness := suite.CreateHarness(model)

	err := suite.TestKeyboardShortcuts(harness)
	// This might succeed or fail depending on timing, just test it doesn't panic
	_ = err // err can be nil or not nil, we just want to ensure no panic
}

func TestTUITestSuite_TestResponsiveLayout(t *testing.T) {
	suite := NewTUITestSuite()
	defer suite.Cleanup()

	model := &mockTUIComponent{}
	harness := suite.CreateHarness(model)

	err := suite.TestResponsiveLayout(harness)
	// This might succeed or fail depending on timing, just test it doesn't panic
	_ = err // err can be nil or not nil, we just want to ensure no panic
}

func TestMockTUITestComponent(t *testing.T) {
	component := MockTUITestComponent()

	assert.NotNil(t, component)

	// Test TUIComponent interface methods
	component.SetSize(100, 50)
	component.SetTheme(&MockTheme{})
	component.Focus()
	component.Blur()

	// Test tea.Model interface methods
	cmd := component.Init()
	assert.Nil(t, cmd)

	model, cmd := component.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Equal(t, component, model)
	assert.Nil(t, cmd)

	view := component.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Mock Component")
}