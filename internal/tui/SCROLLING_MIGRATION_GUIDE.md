# Scrolling System Migration Guide

This guide explains how to migrate from the old pager system to the new standardized scrolling system in NetTraceX TUI.

## Overview

The NetTraceX TUI has been enhanced with a standardized scrolling system that provides consistent behavior across all components. The new system includes:

- **ScrollableList Interface**: Unified interface for scrollable components with selection
- **StandardScrollPager**: New implementation providing consistent scroll behavior
- **ScrollableView**: Enhanced viewport-based scrolling with ScrollableList support
- **Backward Compatibility**: Existing Pager components continue to work unchanged

## Migration Options

### Option 1: No Changes Required (Recommended for Existing Code)

Your existing code using `Pager` will continue to work without any changes:

```go
// This continues to work as before
pager := NewPager()
pager.SetContent("Your content here")
pager.SetSize(width, height)
```

### Option 2: Use PagerScrollableAdapter for ScrollableList Compatibility

If you need ScrollableList interface compatibility with existing Pager:

```go
// Wrap existing pager with adapter
pager := NewPager()
adapter := NewPagerScrollableAdapter(pager)

// Now you can use ScrollableList methods
items := []ScrollableItem{
    NewStringScrollableItem("Line 1", "1"),
    NewStringScrollableItem("Line 2", "2"),
}
adapter.SetItems(items)

// Use standard navigation methods
adapter.MoveDown()
adapter.MoveUp()
adapter.PageDown()
```

### Option 3: Migrate to StandardScrollPager (Recommended for New Code)

For new components, use the StandardScrollPager:

```go
// Create new scroll pager
scrollPager := NewStandardScrollPager()
scrollPager.SetSize(width, height)

// Add scrollable items
items := []ScrollableItem{
    NewNavigationItemScrollable(navItem1),
    NewNavigationItemScrollable(navItem2),
}
scrollPager.SetItems(items)

// Use in your component
func (m *MyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.scrollPager, cmd = m.scrollPager.Update(msg)
    return m, cmd
}

func (m *MyModel) View() string {
    return m.scrollPager.View()
}
```

### Option 4: Use Enhanced ScrollableView

For viewport-based scrolling with optional ScrollableList support:

```go
// Create scrollable view
view := NewScrollableView()
view.SetSize(width, height)

// Option A: Use with string content (legacy mode)
view.SetContent("Your string content here")

// Option B: Use with ScrollableList (new mode)
items := []ScrollableItem{
    NewStringScrollableItem("Item 1", "1"),
    NewStringScrollableItem("Item 2", "2"),
}
view.SetItems(items)

// The view automatically handles both modes
```

## Component-Specific Migration

### NavigationModel

Already migrated to use StandardScrollPager. No changes needed.

### HelpModel

Already migrated to use StandardScrollPager. No changes needed.

### ResultViewModel

Currently uses the old Pager. Migration options:

1. **Keep as-is** (no changes needed)
2. **Wrap with adapter**:
   ```go
   // In NewResultViewModel()
   pager := NewPager()
   adapter := NewPagerScrollableAdapter(pager)
   return &ResultViewModel{
       pager: adapter, // Change type to ScrollableList
   }
   ```
3. **Full migration to StandardScrollPager**:
   ```go
   // Replace pager field with scrollPager
   scrollPager := NewStandardScrollPager()
   // Convert content to ScrollableItems
   ```

## Creating Custom ScrollableItems

To create custom scrollable items, implement the ScrollableItem interface:

```go
type MyCustomItem struct {
    content string
    id      string
    height  int
}

func (m *MyCustomItem) Render(width int, selected bool, theme domain.Theme) string {
    style := ""
    if selected {
        style = theme.GetColor("selected_bg") // Apply selection styling
    }
    return style + m.content
}

func (m *MyCustomItem) GetHeight() int {
    return m.height
}

func (m *MyCustomItem) IsSelectable() bool {
    return true
}

func (m *MyCustomItem) GetID() string {
    return m.id
}
```

## Key Benefits of Migration

1. **Consistency**: All scrollable components behave the same way
2. **Selection Support**: Built-in selection with visual feedback
3. **Smart Scrolling**: Content automatically scrolls to keep selection visible
4. **Keyboard Navigation**: Standardized key bindings (arrows, page up/down, home/end)
5. **Theme Integration**: Consistent styling across all scrollable components
6. **Testing**: Better testability with standardized interfaces

## Backward Compatibility Guarantees

- All existing `Pager` usage continues to work unchanged
- No breaking changes to existing APIs
- `ScrollableView` maintains both string content and ScrollableList modes
- Migration can be done incrementally, component by component

## Performance Considerations

- The new system is designed to be as performant as the old system
- `PagerScrollableAdapter` adds minimal overhead
- `StandardScrollPager` is optimized for large lists with viewport rendering
- Memory usage is similar to the old system

## Testing Your Migration

Use the provided test utilities to verify your migration:

```go
func TestMyScrollableComponent(t *testing.T) {
    component := NewMyComponent()
    
    // Test ScrollableList interface
    var _ ScrollableList = component
    
    // Test navigation
    component.AddItem(NewStringScrollableItem("Test", "1"))
    if !component.MoveDown() {
        t.Error("Expected navigation to work")
    }
}
```

## Getting Help

If you encounter issues during migration:

1. Check that your component implements the correct interfaces
2. Verify that ScrollableItems render correctly
3. Test navigation behavior with different content sizes
4. Use the provided test utilities to validate behavior

The migration is designed to be gradual and non-breaking. Start with the adapter approach for existing components and use the new StandardScrollPager for new development.