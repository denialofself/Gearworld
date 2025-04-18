# Standalone Map Editor Design

## Overview

A standalone map editor built with Ebiten that can create and edit map prefabs for the main game. The editor will be a separate project that shares only the tile definitions and prefab format with the main game.

## Project Structure

```
map_editor/
├── assets/
│   ├── tilesets/
│   └── prefabs/
├── cmd/
│   └── editor/
│       └── main.go
├── internal/
│   ├── editor/
│   │   ├── state.go
│   │   ├── tools.go
│   │   └── layers.go
│   ├── ui/
│   │   ├── toolbar.go
│   │   ├── layer_panel.go
│   │   └── tile_picker.go
│   ├── renderer/
│   │   ├── map_renderer.go
│   │   └── grid_renderer.go
│   └── prefab/
│       ├── loader.go
│       └── saver.go
└── go.mod
```

## Core Components

### Editor State

```go
type EditorState struct {
    CurrentTool     EditorTool
    CurrentLayer    EditorLayer
    SelectedTile    string  // Tile ID from tileset
    SelectedPrefab  string
    GridVisible     bool
    SnapToGrid      bool
    CurrentMap      *MapData
    History         []EditorAction
    HistoryIndex    int
    Camera          Camera
    UIState         UIState
}

type MapData struct {
    Width       int
    Height      int
    Tiles       [][]string
    Features    []Feature
    Entities    []Entity
    Transitions []Transition
}

type Camera struct {
    X, Y        float64
    Zoom        float64
    TargetX, TargetY float64
}
```

### UI Components

```go
type Toolbar struct {
    Tools       []ToolButton
    ActiveTool  EditorTool
    Position    image.Rectangle
}

type LayerPanel struct {
    Layers      []LayerButton
    ActiveLayer EditorLayer
    Position    image.Rectangle
}

type TilePicker struct {
    Tileset     *ebiten.Image
    TileSize    int
    Selected    string
    Position    image.Rectangle
    Scroll      int
}
```

## Main Editor Loop

```go
func (g *Game) Update() error {
    // Handle input
    if err := g.handleInput(); err != nil {
        return err
    }

    // Update camera
    g.updateCamera()

    // Update UI
    g.updateUI()

    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    // Clear screen
    screen.Fill(color.RGBA{40, 40, 40, 255})

    // Draw map
    g.drawMap(screen)

    // Draw grid if enabled
    if g.State.GridVisible {
        g.drawGrid(screen)
    }

    // Draw UI
    g.drawUI(screen)

    // Draw tool preview
    g.drawToolPreview(screen)
}
```

## Input Handling

```go
func (g *Game) handleInput() error {
    // Camera movement
    if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
        g.State.Camera.X += 5
    }
    if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
        g.State.Camera.X -= 5
    }
    // ... other camera controls

    // Tool shortcuts
    if inpututil.IsKeyJustPressed(ebiten.KeyB) {
        g.State.CurrentTool = ToolBrush
    }
    if inpututil.IsKeyJustPressed(ebiten.KeyE) {
        g.State.CurrentTool = ToolEraser
    }
    // ... other tool shortcuts

    // Mouse input
    if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
        x, y := g.screenToWorld(ebiten.CursorPosition())
        g.handleToolUse(x, y)
    }

    return nil
}
```

## Prefab Management

```go
func (g *Game) savePrefab(name string, area image.Rectangle) error {
    prefab := MapPrefab{
        Name: name,
        Width: area.Dx(),
        Height: area.Dy(),
        Tiles: make([][]string, area.Dy()),
    }

    // Copy tiles
    for y := 0; y < area.Dy(); y++ {
        prefab.Tiles[y] = make([]string, area.Dx())
        for x := 0; x < area.Dx(); x++ {
            prefab.Tiles[y][x] = g.State.CurrentMap.Tiles[area.Min.Y+y][area.Min.X+x]
        }
    }

    // Save to JSON
    data, err := json.MarshalIndent(prefab, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(fmt.Sprintf("assets/prefabs/%s.json", name), data, 0644)
}

func (g *Game) loadPrefab(name string) (*MapPrefab, error) {
    data, err := os.ReadFile(fmt.Sprintf("assets/prefabs/%s.json", name))
    if err != nil {
        return nil, err
    }

    var prefab MapPrefab
    if err := json.Unmarshal(data, &prefab); err != nil {
        return nil, err
    }

    return &prefab, nil
}
```

## UI Implementation

```go
func (g *Game) drawUI(screen *ebiten.Image) {
    // Draw toolbar
    g.drawToolbar(screen)

    // Draw layer panel
    g.drawLayerPanel(screen)

    // Draw tile picker
    g.drawTilePicker(screen)

    // Draw status bar
    g.drawStatusBar(screen)
}

func (g *Game) drawToolbar(screen *ebiten.Image) {
    for _, tool := range g.UI.Toolbar.Tools {
        op := &ebiten.DrawImageOptions{}
        op.GeoM.Translate(float64(tool.Position.Min.X), float64(tool.Position.Min.Y))
        
        // Highlight active tool
        if tool.Tool == g.State.CurrentTool {
            screen.DrawImage(g.Assets.ToolHighlight, op)
        }
        
        screen.DrawImage(tool.Icon, op)
    }
}
```

## Camera System

```go
func (g *Game) updateCamera() {
    // Smooth camera movement
    g.State.Camera.X += (g.State.Camera.TargetX - g.State.Camera.X) * 0.1
    g.State.Camera.Y += (g.State.Camera.TargetY - g.State.Camera.Y) * 0.1

    // Zoom limits
    g.State.Camera.Zoom = math.Max(0.25, math.Min(4.0, g.State.Camera.Zoom))
}

func (g *Game) worldToScreen(x, y float64) (float64, float64) {
    screenX := (x - g.State.Camera.X) * g.State.Camera.Zoom
    screenY := (y - g.State.Camera.Y) * g.State.Camera.Zoom
    return screenX, screenY
}

func (g *Game) screenToWorld(x, y int) (float64, float64) {
    worldX := float64(x)/g.State.Camera.Zoom + g.State.Camera.X
    worldY := float64(y)/g.State.Camera.Zoom + g.State.Camera.Y
    return worldX, worldY
}
```

## Design Benefits

1. **Independence**: Completely separate from main game
2. **Focused Development**: Can evolve independently
3. **Simpler Code**: No need to handle game logic
4. **Better Performance**: Dedicated to editing tasks
5. **Easier Distribution**: Can be packaged separately

## Features

1. **Visual Editing**:
   - Tile-based map editing
   - Layer-based editing
   - Grid snapping
   - Camera controls

2. **UI Elements**:
   - Toolbar with common tools
   - Layer panel
   - Tile picker
   - Status bar

3. **File Management**:
   - Save/load prefabs
   - Import/export maps
   - Recent files list

4. **Editing Tools**:
   - Brush tool
   - Eraser tool
   - Fill tool
   - Selection tool
   - Prefab tool

## Future Extensions

1. **Advanced Features**:
   - Auto-tiling
   - Tile variations
   - Custom properties
   - Scripted behaviors

2. **UI Improvements**:
   - Customizable layouts
   - Theme support
   - Keyboard shortcuts
   - Tool presets

3. **Integration**:
   - Live preview
   - Asset management
   - Version control
   - Collaboration tools 