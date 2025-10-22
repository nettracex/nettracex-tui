// Package tui contains core scrolling interfaces and types for standardized scroll behavior
package tui

import (
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// ScrollableList defines the contract for scrollable content with selection
// This interface extends TUIComponent to provide unified scroll behavior
type ScrollableList interface {
	domain.TUIComponent
	
	// Content management
	SetItems(items []ScrollableItem)
	GetItems() []ScrollableItem
	AddItem(item ScrollableItem)
	RemoveItem(index int)
	
	// Selection and scrolling
	GetSelected() int
	SetSelected(index int)
	GetScrollPosition() ScrollPosition
	ScrollToItem(index int)
	
	// Navigation
	MoveUp() bool
	MoveDown() bool
	PageUp() bool
	PageDown() bool
	Home() bool
	End() bool
	
	// Viewport management
	GetVisibleRange() (start, end int)
	IsItemVisible(index int) bool
}

// ScrollableItem defines the contract for items that can be rendered in a scrollable list
type ScrollableItem interface {
	// Render returns the string representation of the item
	// width: available width for rendering
	// selected: whether this item is currently selected
	// theme: theme to apply for styling
	Render(width int, selected bool, theme domain.Theme) string
	
	// GetHeight returns the number of lines this item occupies when rendered
	GetHeight() int
	
	// IsSelectable returns true if this item can be selected by the user
	IsSelectable() bool
	
	// GetID returns a unique identifier for this item
	GetID() string
}

// ScrollPosition tracks the current scroll and selection state
type ScrollPosition struct {
	// SelectedIndex is the index of the currently selected item
	SelectedIndex int
	
	// TopVisible is the index of the first visible item in the viewport
	TopVisible int
	
	// ViewportHeight is the number of items that can fit in the viewport
	ViewportHeight int
}

// ScrollableContent represents scrollable content with metadata
type ScrollableContent struct {
	// Items contains all scrollable items
	Items []ScrollableItem
	
	// Position tracks current scroll and selection state
	Position ScrollPosition
	
	// ShowIndicators controls whether scroll indicators are displayed
	ShowIndicators bool
	
	// Theme for styling the content
	Theme domain.Theme
	
	// KeyMap for navigation key bindings
	KeyMap KeyMap
}

// NewScrollPosition creates a new ScrollPosition with default values
func NewScrollPosition() ScrollPosition {
	return ScrollPosition{
		SelectedIndex:  0,
		TopVisible:     0,
		ViewportHeight: 1,
	}
}

// NewScrollableContent creates a new ScrollableContent with default settings
func NewScrollableContent() *ScrollableContent {
	return &ScrollableContent{
		Items:          make([]ScrollableItem, 0),
		Position:       NewScrollPosition(),
		ShowIndicators: true,
		KeyMap:         DefaultKeyMap(),
	}
}

// IsValid returns true if the scroll position is valid for the given content
func (sp ScrollPosition) IsValid(itemCount int) bool {
	if itemCount == 0 {
		return sp.SelectedIndex == 0 && sp.TopVisible == 0
	}
	
	return sp.SelectedIndex >= 0 && 
		   sp.SelectedIndex < itemCount && 
		   sp.TopVisible >= 0 && 
		   sp.TopVisible < itemCount &&
		   sp.ViewportHeight > 0
}

// EnsureSelectionVisible adjusts TopVisible to ensure the selected item is visible
func (sp *ScrollPosition) EnsureSelectionVisible(itemCount int) {
	if itemCount == 0 {
		sp.SelectedIndex = 0
		sp.TopVisible = 0
		return
	}
	
	// Clamp selected index to valid range
	if sp.SelectedIndex < 0 {
		sp.SelectedIndex = 0
	}
	if sp.SelectedIndex >= itemCount {
		sp.SelectedIndex = itemCount - 1
	}
	
	// Adjust TopVisible to keep selection visible
	if sp.SelectedIndex < sp.TopVisible {
		// Selection is above viewport, scroll up
		sp.TopVisible = sp.SelectedIndex
	} else if sp.SelectedIndex >= sp.TopVisible+sp.ViewportHeight {
		// Selection is below viewport, scroll down
		sp.TopVisible = sp.SelectedIndex - sp.ViewportHeight + 1
	}
	
	// Ensure TopVisible is within bounds
	maxTopVisible := itemCount - sp.ViewportHeight
	if maxTopVisible < 0 {
		maxTopVisible = 0
	}
	if sp.TopVisible > maxTopVisible {
		sp.TopVisible = maxTopVisible
	}
	if sp.TopVisible < 0 {
		sp.TopVisible = 0
	}
}

// CanScrollUp returns true if content can be scrolled up
func (sp ScrollPosition) CanScrollUp() bool {
	return sp.TopVisible > 0
}

// CanScrollDown returns true if content can be scrolled down
func (sp ScrollPosition) CanScrollDown(itemCount int) bool {
	return sp.TopVisible+sp.ViewportHeight < itemCount
}

// GetVisibleRange returns the start and end indices of visible items
func (sp ScrollPosition) GetVisibleRange(itemCount int) (start, end int) {
	start = sp.TopVisible
	end = sp.TopVisible + sp.ViewportHeight
	
	if start < 0 {
		start = 0
	}
	if end > itemCount {
		end = itemCount
	}
	if start > itemCount {
		start = itemCount
	}
	
	return start, end
}

// IsItemVisible returns true if the item at the given index is visible
func (sp ScrollPosition) IsItemVisible(index int) bool {
	start, end := sp.GetVisibleRange(index + 1) // +1 because we need at least index+1 items
	return index >= start && index < end
}

// AddItem adds an item to the scrollable content
func (sc *ScrollableContent) AddItem(item ScrollableItem) {
	sc.Items = append(sc.Items, item)
}

// RemoveItem removes an item at the specified index
func (sc *ScrollableContent) RemoveItem(index int) {
	if index < 0 || index >= len(sc.Items) {
		return
	}
	
	sc.Items = append(sc.Items[:index], sc.Items[index+1:]...)
	
	// Adjust selection if necessary
	if sc.Position.SelectedIndex >= len(sc.Items) && len(sc.Items) > 0 {
		sc.Position.SelectedIndex = len(sc.Items) - 1
	} else if len(sc.Items) == 0 {
		sc.Position.SelectedIndex = 0
	}
	
	// Ensure selection remains visible
	sc.Position.EnsureSelectionVisible(len(sc.Items))
}

// SetItems replaces all items in the scrollable content
func (sc *ScrollableContent) SetItems(items []ScrollableItem) {
	sc.Items = items
	sc.Position.SelectedIndex = 0
	sc.Position.TopVisible = 0
	sc.Position.EnsureSelectionVisible(len(sc.Items))
}

// GetSelectedItem returns the currently selected item, or nil if none
func (sc *ScrollableContent) GetSelectedItem() ScrollableItem {
	if sc.Position.SelectedIndex >= 0 && sc.Position.SelectedIndex < len(sc.Items) {
		return sc.Items[sc.Position.SelectedIndex]
	}
	return nil
}

// SetViewportHeight updates the viewport height and adjusts scroll position
func (sc *ScrollableContent) SetViewportHeight(height int) {
	if height < 1 {
		height = 1
	}
	sc.Position.ViewportHeight = height
	sc.Position.EnsureSelectionVisible(len(sc.Items))
}