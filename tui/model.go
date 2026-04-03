package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"vmtui/vm"
)

// ---- Tea messages ----

type statusMsg struct {
	text  string
	isErr bool
}

type vmStartedMsg struct {
	id  string
	pid int
}

type vmErrorMsg struct {
	id  string
	err string
}

type tickMsg time.Time

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Every(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ---- Screen interface (State Pattern) ----

// Screen encapsulates the Update/View behavior of one TUI screen.
// Each concrete screen type receives RootModel by value and returns a modified copy,
// matching Bubble Tea's functional update semantics.
type Screen interface {
	Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd)
	View(m RootModel) string
}

// ---- root model ----

type RootModel struct {
	screen  Screen
	list    ListModel
	form    FormModel
	catalog CatalogModel
	store   *vm.Store
	pids    *vm.PIDStore
	width   int
	height  int

	installTarget *vm.VMConfig
	isoInput      string
	isoErr        string
	isoChoices    []string
	isoChoiceIdx  int

	editingVMID  string
	deleteTarget *vm.VMConfig

	logTitle   string
	logContent string
	logErr     string

	assetDisks             []assetEntry
	assetISOs              []assetEntry
	assetErr               string
	assetCursor            int
	assetPendingDeletePath string
}

type assetEntry struct {
	Kind       string
	Title      string
	Meta       string
	Path       string
	Exists     bool
	OwnerIDs   []string
	OwnerNames []string
}

func NewRootModel(store *vm.Store, pids *vm.PIDStore) RootModel {
	m := RootModel{
		store: store,
		pids:  pids,
	}
	m.list = NewListModel(store, pids)
	m.screen = ListScreen{}
	return m
}

func (m RootModel) Init() tea.Cmd {
	return tickEvery(2 * time.Second)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}

	if _, ok := msg.(tickMsg); ok {
		m.list = m.list.RefreshStatus(m.pids)
		return m, tickEvery(2 * time.Second)
	}

	return m.screen.Update(m, msg)
}

func (m RootModel) View() string {
	return m.screen.View(m)
}
