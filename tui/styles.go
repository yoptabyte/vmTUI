package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/lipgloss"
)

type colorPalette struct {
	Bg         string `json:"bg"`
	Panel      string `json:"panel"`
	PanelAlt   string `json:"panelAlt"`
	Border     string `json:"border"`
	Accent     string `json:"accent"`
	AccentSoft string `json:"accentSoft"`
	Success    string `json:"success"`
	Danger     string `json:"danger"`
	Warning    string `json:"warning"`
	Text       string `json:"text"`
	Subtext    string `json:"subtext"`
	Muted      string `json:"muted"`
}

var colors colorPalette
var styles struct {
	App          lipgloss.Style
	Panel        lipgloss.Style
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	Section      lipgloss.Style
	Selected     lipgloss.Style
	Normal       lipgloss.Style
	Dim          lipgloss.Style
	StatusBar    lipgloss.Style
	Error        lipgloss.Style
	Success      lipgloss.Style
	Warning      lipgloss.Style
	BadgeRunning lipgloss.Style
	BadgeStopped lipgloss.Style
	Row          lipgloss.Style
	RowSelected  lipgloss.Style
	Card         lipgloss.Style
}

func init() {
	colors = loadColors()
	initStyles()
}

func loadColors() colorPalette {
	defaultColors := colorPalette{
		Bg:         "#11111B",
		Panel:      "#1E1E2E",
		PanelAlt:   "#181825",
		Border:     "#313244",
		Accent:     "#CBA6F7",
		AccentSoft: "#B4BEFE",
		Success:    "#A6E3A1",
		Danger:     "#F38BA8",
		Warning:    "#F9E2AF",
		Text:       "#CDD6F4",
		Subtext:    "#A6ADC8",
		Muted:      "#7F849C",
	}

	_, filename, _, _ := runtime.Caller(0)
	configPath := filepath.Join(filepath.Dir(filename), "colors.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultColors
	}

	var loaded colorPalette
	if err := json.Unmarshal(data, &loaded); err != nil {
		return defaultColors
	}

	return loaded
}

func c(hex string) lipgloss.TerminalColor {
	return lipgloss.Color(hex)
}

func initStyles() {
	styles.App = lipgloss.NewStyle().
		Background(c(colors.Bg)).
		Foreground(c(colors.Text))

	styles.Panel = lipgloss.NewStyle().
		Background(c(colors.Panel)).
		Foreground(c(colors.Text)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(colors.Border)).
		Padding(1, 2)

	styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(c(colors.Accent)).
		Background(c(colors.Panel))

	styles.Subtitle = lipgloss.NewStyle().
		Foreground(c(colors.Subtext)).
		Background(c(colors.Panel))

	styles.Section = lipgloss.NewStyle().
		Bold(true).
		Foreground(c(colors.AccentSoft)).
		Background(c(colors.Panel))

	styles.Selected = lipgloss.NewStyle().
		Bold(true).
		Foreground(c(colors.Bg)).
		Background(c(colors.Accent)).
		Padding(0, 1)

	styles.Normal = lipgloss.NewStyle().
		Foreground(c(colors.Text)).
		Background(c(colors.Panel))

	styles.Dim = lipgloss.NewStyle().
		Foreground(c(colors.Muted)).
		Background(c(colors.Panel))

	styles.StatusBar = lipgloss.NewStyle().
		Foreground(c(colors.Subtext)).
		Background(c(colors.Panel)).
		Padding(0, 1)

	styles.Error = lipgloss.NewStyle().
		Foreground(c(colors.Danger)).
		Background(c(colors.Panel))

	styles.Success = lipgloss.NewStyle().
		Foreground(c(colors.Success)).
		Background(c(colors.Panel))

	styles.Warning = lipgloss.NewStyle().
		Foreground(c(colors.Warning)).
		Background(c(colors.Panel))

	styles.BadgeRunning = lipgloss.NewStyle().
		Bold(true).
		Foreground(c(colors.Bg)).
		Background(c(colors.Success)).
		Padding(0, 1)

	styles.BadgeStopped = lipgloss.NewStyle().
		Foreground(c(colors.Muted)).
		Background(c(colors.Panel)).
		Padding(0, 1)

	styles.Row = lipgloss.NewStyle().
		Background(c(colors.Panel)).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(c(colors.Border)).
		Padding(0, 1)

	styles.RowSelected = styles.Row.Copy().
		BorderForeground(c(colors.AccentSoft))

	styles.Card = lipgloss.NewStyle().
		Background(c(colors.Panel)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c(colors.Border)).
		Padding(1, 1)
}

// Backward-compatible aliases
var (
	styleApp          = &styles.App
	stylePanel        = &styles.Panel
	styleTitle        = &styles.Title
	styleSubtitle     = &styles.Subtitle
	styleSection      = &styles.Section
	styleSelected     = &styles.Selected
	styleNormal       = &styles.Normal
	styleDim          = &styles.Dim
	styleStatusBar    = &styles.StatusBar
	styleError        = &styles.Error
	styleSuccess      = &styles.Success
	styleWarning      = &styles.Warning
	styleBadgeRunning = &styles.BadgeRunning
	styleBadgeStopped = &styles.BadgeStopped
	styleRow          = &styles.Row
	styleRowSelected  = &styles.RowSelected
	styleCard         = &styles.Card
)

func renderScreen(width, height int, content string) string {
	panelWidth := 92
	panelHeight := 24
	if width > 0 {
		panelWidth = max(24, width-4)
	}
	if height > 0 {
		panelHeight = max(8, height-2)
	}

	panel := stylePanel.Width(panelWidth).Height(panelHeight).Render(content)
	if width <= 0 || height <= 0 {
		return panel
	}

	canvasWidth := max(width, lipgloss.Width(panel))
	canvasHeight := max(height, lipgloss.Height(panel))
	placed := lipgloss.Place(
		canvasWidth,
		canvasHeight,
		lipgloss.Left,
		lipgloss.Top,
		lipgloss.NewStyle().Padding(1, 2).Render(panel),
	)
	return styleApp.Width(canvasWidth).Height(canvasHeight).Render(placed)
}

func panelContentSize(width, height int) (int, int) {
	panelWidth := 92
	panelHeight := 24
	if width > 0 {
		panelWidth = max(24, width-4)
	}
	if height > 0 {
		panelHeight = max(8, height-2)
	}

	contentWidth := max(12, panelWidth-stylePanel.GetHorizontalFrameSize())
	contentHeight := max(4, panelHeight-stylePanel.GetVerticalFrameSize())
	return contentWidth, contentHeight
}
