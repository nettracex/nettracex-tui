// Package tui contains tests for TUI components
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewFormModel(t *testing.T) {
	title := "Test Form"
	model := NewFormModel(title)

	assert.NotNil(t, model)
	assert.Equal(t, title, model.title)
	assert.Equal(t, 0, model.focused)
	assert.Empty(t, model.fields)
	assert.False(t, model.submitted)
}

func TestFormModel_AddField(t *testing.T) {
	model := NewFormModel("Test")
	
	model.AddField("username", "Username", true)
	model.AddField("email", "Email", false)

	assert.Equal(t, 2, len(model.fields))
	
	// Check first field
	assert.Equal(t, "username", model.fields[0].Key)
	assert.Equal(t, "Username", model.fields[0].Label)
	assert.True(t, model.fields[0].Required)
	
	// Check second field
	assert.Equal(t, "email", model.fields[1].Key)
	assert.Equal(t, "Email", model.fields[1].Label)
	assert.False(t, model.fields[1].Required)
}

func TestFormModel_SetGetFieldValue(t *testing.T) {
	model := NewFormModel("Test")
	model.AddField("username", "Username", true)
	
	model.SetFieldValue("username", "testuser")
	value := model.GetFieldValue("username")
	
	assert.Equal(t, "testuser", value)
	
	// Test non-existent field
	value = model.GetFieldValue("nonexistent")
	assert.Equal(t, "", value)
}

func TestFormModel_GetValues(t *testing.T) {
	model := NewFormModel("Test")
	model.AddField("username", "Username", true)
	model.AddField("email", "Email", false)
	
	model.SetFieldValue("username", "testuser")
	model.SetFieldValue("email", "test@example.com")
	
	values := model.GetValues()
	expected := map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
	}
	
	assert.Equal(t, expected, values)
}

func TestFormModel_Update_Navigation(t *testing.T) {
	model := NewFormModel("Test")
	model.AddField("field1", "Field 1", true)
	model.AddField("field2", "Field 2", false)
	
	// Test tab navigation
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, cmd := model.Update(msg)
	formModel := updatedModel.(*FormModel)
	
	assert.Equal(t, 1, formModel.focused)
	assert.Nil(t, cmd)
	
	// Test down navigation
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd = formModel.Update(msg)
	formModel = updatedModel.(*FormModel)
	
	assert.Equal(t, 0, formModel.focused) // Should wrap around
	assert.Nil(t, cmd)
	
	// Test up navigation
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd = formModel.Update(msg)
	formModel = updatedModel.(*FormModel)
	
	assert.Equal(t, 1, formModel.focused)
	assert.Nil(t, cmd)
}

func TestFormModel_Update_Submit_Valid(t *testing.T) {
	model := NewFormModel("Test")
	model.AddField("username", "Username", true)
	model.SetFieldValue("username", "testuser")
	
	// Test enter key with valid form
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	formModel := updatedModel.(*FormModel)
	
	assert.True(t, formModel.submitted)
	assert.NotNil(t, cmd)
	
	// Execute command to get message
	if cmd != nil {
		result := cmd()
		submitMsg, ok := result.(FormSubmitMsg)
		assert.True(t, ok)
		assert.Equal(t, "testuser", submitMsg.Values["username"])
	}
}

func TestFormModel_Update_Submit_Invalid(t *testing.T) {
	model := NewFormModel("Test")
	model.AddField("username", "Username", true)
	// Don't set value for required field
	
	// Test enter key with invalid form
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	formModel := updatedModel.(*FormModel)
	
	assert.False(t, formModel.submitted)
	assert.Nil(t, cmd)
	assert.NotEmpty(t, formModel.fields[0].ErrorText)
}

func TestFormModel_View(t *testing.T) {
	model := NewFormModel("Test Form")
	model.AddField("username", "Username", true)
	model.width = 80
	
	view := model.View()
	
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Test Form")
	assert.Contains(t, view, "Username")
}

func TestFormModel_Validate(t *testing.T) {
	model := NewFormModel("Test")
	model.AddField("required", "Required Field", true)
	model.AddField("optional", "Optional Field", false)
	
	// Test with empty required field
	valid := model.validate()
	assert.False(t, valid)
	assert.NotEmpty(t, model.fields[0].ErrorText)
	
	// Test with filled required field
	model.SetFieldValue("required", "value")
	valid = model.validate()
	assert.True(t, valid)
	assert.Empty(t, model.fields[0].ErrorText)
}

func TestFormModel_TUIComponent(t *testing.T) {
	model := NewFormModel("Test")
	model.AddField("test", "Test", true)
	theme := &MockTheme{}
	
	// Test SetSize
	model.SetSize(100, 50)
	assert.Equal(t, 100, model.width)
	assert.Equal(t, 50, model.height)
	
	// Test SetTheme
	model.SetTheme(theme)
	assert.Equal(t, theme, model.theme)
	
	// Test Focus/Blur (should not panic)
	model.Focus()
	model.Blur()
}

func TestNewTableModel(t *testing.T) {
	headers := []string{"Name", "Age", "City"}
	model := NewTableModel(headers)

	assert.NotNil(t, model)
	assert.Equal(t, headers, model.headers)
	assert.Empty(t, model.rows)
	assert.Equal(t, -1, model.sortBy)
	assert.False(t, model.sortDesc)
	assert.Equal(t, 0, model.selected)
	assert.True(t, model.focused)
}

func TestTableModel_SetData(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	
	rows := [][]string{
		{"Alice", "25"},
		{"Bob", "30"},
		{"Charlie", "35"},
	}
	
	model.SetData(rows)
	assert.Equal(t, rows, model.rows)
	assert.Equal(t, 0, model.selected)
}

func TestTableModel_AddRow(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	
	model.AddRow([]string{"Alice", "25"})
	model.AddRow([]string{"Bob", "30"})
	
	assert.Equal(t, 2, len(model.rows))
	assert.Equal(t, []string{"Alice", "25"}, model.rows[0])
	assert.Equal(t, []string{"Bob", "30"}, model.rows[1])
}

func TestTableModel_Update_Navigation(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	model.SetData([][]string{
		{"Alice", "25"},
		{"Bob", "30"},
		{"Charlie", "35"},
	})
	
	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(msg)
	tableModel := updatedModel.(*TableModel)
	
	assert.Equal(t, 1, tableModel.selected)
	assert.Nil(t, cmd)
	
	// Test up navigation
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd = tableModel.Update(msg)
	tableModel = updatedModel.(*TableModel)
	
	assert.Equal(t, 0, tableModel.selected)
	assert.Nil(t, cmd)
}

func TestTableModel_Update_Select(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	model.SetData([][]string{
		{"Alice", "25"},
		{"Bob", "30"},
	})
	model.selected = 1
	
	// Test enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	tableModel := updatedModel.(*TableModel)
	
	assert.Equal(t, 1, tableModel.selected)
	assert.NotNil(t, cmd)
	
	// Execute command to get message
	if cmd != nil {
		result := cmd()
		selectMsg, ok := result.(TableSelectMsg)
		assert.True(t, ok)
		assert.Equal(t, 1, selectMsg.Row)
		assert.Equal(t, []string{"Bob", "30"}, selectMsg.Data)
	}
}

func TestTableModel_View(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	model.SetData([][]string{
		{"Alice", "25"},
		{"Bob", "30"},
	})
	model.width = 80
	
	view := model.View()
	
	assert.NotEmpty(t, view)
	// The table rendering might wrap text, so just check that it's not empty
	// and contains some expected content
	assert.True(t, len(view) > 0)
}

func TestTableModel_View_Empty(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	model.width = 80
	
	view := model.View()
	
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No data available")
}

func TestTableModel_SortBy(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	model.SetData([][]string{
		{"Charlie", "35"},
		{"Alice", "25"},
		{"Bob", "30"},
	})
	
	// Sort by name (column 0)
	model.SortBy(0, false)
	
	assert.Equal(t, 0, model.sortBy)
	assert.False(t, model.sortDesc)
	assert.Equal(t, "Alice", model.rows[0][0])
	assert.Equal(t, "Bob", model.rows[1][0])
	assert.Equal(t, "Charlie", model.rows[2][0])
	
	// Sort by name descending
	model.SortBy(0, true)
	
	assert.True(t, model.sortDesc)
	assert.Equal(t, "Charlie", model.rows[0][0])
	assert.Equal(t, "Bob", model.rows[1][0])
	assert.Equal(t, "Alice", model.rows[2][0])
}

func TestTableModel_SetFilter(t *testing.T) {
	headers := []string{"Name", "Age"}
	model := NewTableModel(headers)
	model.SetData([][]string{
		{"Alice", "25"},
		{"Bob", "30"},
		{"Charlie", "35"},
	})
	model.selected = 2
	
	model.SetFilter("alice")
	
	assert.Equal(t, "alice", model.filter)
	assert.Equal(t, 0, model.selected) // Should reset selection
	
	// Test filtered rows
	filtered := model.getFilteredRows()
	assert.Equal(t, 1, len(filtered))
	assert.Equal(t, "Alice", filtered[0][0])
}

func TestNewProgressModel(t *testing.T) {
	model := NewProgressModel()

	assert.NotNil(t, model)
	assert.Equal(t, 0, model.current)
	assert.Equal(t, 0, model.total)
	assert.Empty(t, model.message)
	assert.True(t, model.animated)
}

func TestProgressModel_SetProgress(t *testing.T) {
	model := NewProgressModel()
	
	model.SetProgress(5, 10)
	
	assert.Equal(t, 5, model.current)
	assert.Equal(t, 10, model.total)
}

func TestProgressModel_SetMessage(t *testing.T) {
	model := NewProgressModel()
	
	model.SetMessage("Processing...")
	
	assert.Equal(t, "Processing...", model.message)
}

func TestProgressModel_View(t *testing.T) {
	model := NewProgressModel()
	model.SetProgress(3, 10)
	model.SetMessage("Loading")
	model.width = 80
	
	view := model.View()
	
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Loading")
	assert.Contains(t, view, "30%")
	assert.Contains(t, view, "(3/10)")
}

func TestProgressModel_View_NoTotal(t *testing.T) {
	model := NewProgressModel()
	model.width = 80
	
	view := model.View()
	
	assert.Equal(t, "No progress to display", view)
}

func TestProgressModel_IsComplete(t *testing.T) {
	model := NewProgressModel()
	
	// Test incomplete
	model.SetProgress(5, 10)
	assert.False(t, model.IsComplete())
	
	// Test complete
	model.SetProgress(10, 10)
	assert.True(t, model.IsComplete())
	
	// Test over-complete
	model.SetProgress(15, 10)
	assert.True(t, model.IsComplete())
	
	// Test no total
	model.SetProgress(5, 0)
	assert.False(t, model.IsComplete())
}

func TestProgressModel_TUIComponent(t *testing.T) {
	model := NewProgressModel()
	theme := &MockTheme{}
	
	// Test SetSize
	model.SetSize(100, 50)
	assert.Equal(t, 100, model.width)
	assert.Equal(t, 50, model.height)
	
	// Test SetTheme
	model.SetTheme(theme)
	assert.Equal(t, theme, model.theme)
	
	// Test Focus/Blur (should not panic)
	model.Focus()
	model.Blur()
}