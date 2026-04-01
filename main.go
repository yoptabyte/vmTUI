package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"vmctl/tui"
	"vmctl/vm"
)

func main() {
	store, err := vm.LoadStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading vm store: %v\n", err)
		os.Exit(1)
	}

	pids := vm.LoadPIDStore()

	model := tui.NewRootModel(store, pids)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
