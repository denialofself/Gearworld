# Render System Refactoring Design

## Overview

This document outlines the proposed refactoring of the render system to improve maintainability, reduce complexity, and prevent cyclic imports. The current render system is contained in a single large file (`render_system.go`) and handles multiple responsibilities including core rendering, UI elements, and state management.

## Goals

1. Reduce file size and complexity
2. Prevent cyclic imports
3. Improve maintainability
4. Separate concerns
5. Make the system more testable
6. Maintain existing functionality

## Proposed Structure

```
systems/
├── render/
│   ├── core/
│   │   ├── render_system.go     # Core rendering system
│   │   ├── camera.go            # Camera functionality
│   │   └── tileset.go           # Tileset handling
│   └── ui/
│       ├── ui_system.go         # UI system (depends on core)
│       ├── inventory_panel.go   # Inventory UI
│       ├── stats_panel.go       # Stats panel
│       ├── message_panel.go     # Message log
│       └── debug_window.go      # Debug window
```

## Component Details

### Core Package

#### RenderSystem (`core/render_system.go`)
- Handles core rendering functionality
- Manages map and entity rendering
- Provides rendering services to UI system
- No knowledge of UI implementation

```go
type RenderSystem struct {
    camera  *Camera
    tileset *Tileset
}

func (s *RenderSystem) DrawMap(screen *ebiten.Image) {}
func (s *RenderSystem) DrawEntities(screen *ebiten.Image) {}
```

#### Camera (`core/camera.go`)
- Handles camera positioning and movement
- Manages viewport calculations
- Provides camera services to render system

```go
type Camera struct {
    x, y int
    targetID ecs.EntityID
}

func (c *Camera) UpdatePosition(x, y int) {}
func (c *Camera) GetViewport() (int, int, int, int) {}
```

#### Tileset (`core/tileset.go`)
- Handles tile loading and drawing
- Manages sprite resources
- Provides tile drawing services

```go
type Tileset struct {
    image *ebiten.Image
    tileSize int
}

func (t *Tileset) DrawTile(screen *ebiten.Image, tileID TileID, x, y int) {}
```

### UI Package

#### UISystem (`ui/ui_system.go`)
- Manages all UI components
- Coordinates UI rendering
- Handles UI state
- Depends on core package

```go
type UISystem struct {
    renderer Renderer
    inventory *InventoryPanel
    stats *StatsPanel
    messages *MessagePanel
    debug *DebugWindow
}

func (u *UISystem) Draw(screen *ebiten.Image) {}
```

#### InventoryPanel (`ui/inventory_panel.go`)
- Handles inventory display
- Manages item selection
- Handles item details view

```go
type InventoryPanel struct {
    selectedItemIndex int
    itemViewMode bool
    items []Item
}

func (p *InventoryPanel) Draw(screen *ebiten.Image) {}
```

#### StatsPanel (`ui/stats_panel.go`)
- Displays player statistics
- Shows equipment
- Handles stat updates

```go
type StatsPanel struct {
    playerStats *Stats
    equipment *Equipment
}

func (p *StatsPanel) Draw(screen *ebiten.Image) {}
```

#### MessagePanel (`ui/message_panel.go`)
- Manages message log
- Handles message display
- Controls message scrolling

```go
type MessagePanel struct {
    messages []Message
    scrollOffset int
}

func (p *MessagePanel) Draw(screen *ebiten.Image) {}
```

#### DebugWindow (`ui/debug_window.go`)
- Handles debug information display
- Manages debug state
- Controls debug visibility

```go
type DebugWindow struct {
    active bool
    scrollOffset int
    messages []string
}

func (w *DebugWindow) Draw(screen *ebiten.Image) {}
```

## Communication Patterns

### Interface-Based Communication

```go
// core/render_system.go
type Renderer interface {
    Draw(screen *ebiten.Image)
    GetCamera() *Camera
    GetTileset() *Tileset
}

// ui/ui_system.go
type UISystem struct {
    renderer Renderer  // Uses interface instead of concrete type
}
```

### Event-Based Communication

```go
// core/events.go
type RenderEvent struct {
    Type string
    Data interface{}
}

// ui/ui_system.go
func (u *UISystem) HandleRenderEvent(event RenderEvent) {
    switch event.Type {
    case "camera_update":
        // Handle camera update
    case "tileset_change":
        // Handle tileset change
    }
}
```

## State Management

### Core State
- Camera position and target
- Tileset resources
- Map and entity rendering state

### UI State
- Inventory selection and view mode
- Stats panel visibility
- Message log state
- Debug window state

## Dependencies

### Core Package
- Depends on:
  - ECS system
  - Configuration
  - Basic utilities

### UI Package
- Depends on:
  - Core package
  - ECS system
  - Configuration

## Implementation Strategy

1. Create new directory structure
2. Move core rendering functionality
3. Extract UI components
4. Implement interfaces
5. Set up event system
6. Update dependencies
7. Test each component

## Testing Strategy

1. Unit Tests
   - Test each component in isolation
   - Mock dependencies
   - Verify rendering output

2. Integration Tests
   - Test component interactions
   - Verify event handling
   - Check state management

3. Performance Tests
   - Measure rendering performance
   - Check memory usage
   - Verify no regressions

## Migration Plan

1. Phase 1: Core System
   - Extract core rendering
   - Move camera and tileset
   - Update dependencies

2. Phase 2: UI System
   - Extract UI components
   - Implement interfaces
   - Set up event system

3. Phase 3: Integration
   - Connect components
   - Verify functionality
   - Test performance

4. Phase 4: Cleanup
   - Remove old code
   - Update documentation
   - Final testing

## Benefits

1. Reduced Complexity
   - Smaller, focused files
   - Clear responsibilities
   - Easier to understand

2. Improved Maintainability
   - Easier to modify components
   - Better code organization
   - Clear dependencies

3. Better Testing
   - Components can be tested independently
   - Easier to mock dependencies
   - Clear testing boundaries

4. No Cyclic Imports
   - Clear dependency hierarchy
   - Unidirectional dependencies
   - Interface-based communication

## Risks and Mitigation

1. Risk: Breaking existing functionality
   - Mitigation: Thorough testing
   - Mitigation: Gradual migration
   - Mitigation: Feature flags

2. Risk: Performance impact
   - Mitigation: Performance testing
   - Mitigation: Optimization
   - Mitigation: Profiling

3. Risk: Complex migration
   - Mitigation: Clear migration plan
   - Mitigation: Documentation
   - Mitigation: Team communication

## Future Considerations

1. Extensibility
   - Adding new UI components
   - Supporting new rendering features
   - Customizing rendering pipeline

2. Performance Optimization
   - Batch rendering
   - Caching
   - Lazy loading

3. Feature Additions
   - New UI elements
   - Advanced rendering effects
   - Custom shaders 