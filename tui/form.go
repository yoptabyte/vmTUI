package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"vmctl/vm"
)

// FormModel is shown when the user presses 'n' to create a new VM.
type FormModel struct {
	form     *huh.Form
	done     bool
	aborted  bool
	result   vm.VMConfig
	fields   *formFields
	title    string
	autoDisk string
}

type formFields struct {
	name        string
	memStr      string
	vcpuStr     string
	diskSizeStr string
	portStr     string
	diskFile    string
	isoPath     string
}

func NewFormModel() FormModel {
	return newFormModel(vm.VMConfig{}, "New Virtual Machine")
}

func NewEditFormModel(cfg vm.VMConfig) FormModel {
	return newFormModel(cfg, "Edit Virtual Machine")
}

func newFormModel(cfg vm.VMConfig, title string) FormModel {
	diskFile := cfg.DiskFile
	if diskFile == "" {
		diskFile = vm.DefaultDiskPath(cfg.Name)
	}

	fields := &formFields{
		name:        cfg.Name,
		memStr:      defaultFormInt(cfg.MemMiB, 4096),
		vcpuStr:     defaultFormInt(cfg.VCPUs, 4),
		diskSizeStr: defaultFormInt(cfg.DiskSizeGiB, 40),
		portStr:     defaultFormInt(cfg.SSHPort, 2226),
		diskFile:    diskFile,
		isoPath:     cfg.ISOPath,
	}

	m := FormModel{
		result: vm.VMConfig{
			ID:          cfg.ID,
			Type:        vm.TypeQEMU,
			ISOPath:     cfg.ISOPath,
			DiskSizeGiB: cfg.DiskSizeGiB,
		},
		fields:   fields,
		title:    title,
		autoDisk: diskFile,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("VM Name").
				Description("Display name for the virtual machine").
				Placeholder("My VM").
				Value(&fields.name).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("name cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("RAM (MiB)").
				Description("Memory in mebibytes (e.g. 4096 = 4 GiB)").
				Value(&fields.memStr).
				Validate(validateInt),

			huh.NewInput().
				Title("vCPUs").
				Description("Number of virtual CPU cores").
				Value(&fields.vcpuStr).
				Validate(validateInt),

			huh.NewInput().
				Title("Disk size (GiB)").
				Description("Size of the new qcow2 disk image").
				Value(&fields.diskSizeStr).
				Validate(validateInt),

			huh.NewInput().
				Title("Disk file path").
				Description("Where to store the .qcow2 file").
				Placeholder("my-vm.qcow2").
				Value(&fields.diskFile).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("disk file path cannot be empty")
					}
					if !strings.HasSuffix(s, ".qcow2") {
						return fmt.Errorf("must end in .qcow2")
					}
					return nil
				}),

			huh.NewInput().
				Title("SSH host port").
				Description("Port on 127.0.0.1 forwarded to guest :22").
				Value(&fields.portStr).
				Validate(validatePort),

			huh.NewInput().
				Title("ISO path").
				Description("Path to the boot/installation ISO image").
				Placeholder("/path/to/image.iso").
				Value(&fields.isoPath).
				Validate(func(s string) error {
					if s != "" && !strings.HasSuffix(strings.ToLower(s), ".iso") {
						return fmt.Errorf("must end in .iso or be empty")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeCatppuccin())

	m.form = form
	return m
}

func (m FormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	if m.done || m.aborted {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.aborted = true
			return m, nil
		}
	}

	f, cmd := m.form.Update(msg)
	m.form = f.(*huh.Form)

	suggestedDisk := vm.DefaultDiskPath(m.fields.name)
	if m.fields.diskFile == "" || m.fields.diskFile == m.autoDisk {
		m.fields.diskFile = suggestedDisk
	}
	m.autoDisk = suggestedDisk

	switch m.form.State {
	case huh.StateCompleted:
		m.done = true
		m.buildResult()
	case huh.StateAborted:
		m.aborted = true
	}

	return m, cmd
}

func (m *FormModel) buildResult() {
	if m.fields == nil {
		return
	}
	m.result = vm.VMConfig{
		ID:          m.result.ID,
		Type:        vm.TypeQEMU,
		Name:        strings.TrimSpace(m.fields.name),
		MemMiB:      mustAtoi(m.fields.memStr),
		VCPUs:       mustAtoi(m.fields.vcpuStr),
		DiskFile:    vm.ResolvePath(m.fields.diskFile),
		DiskSizeGiB: mustAtoi(m.fields.diskSizeStr),
		SSHPort:     mustAtoi(m.fields.portStr),
		ISOPath:     m.fields.isoPath,
	}
}

func (m FormModel) View() string {
	if m.aborted {
		return styleError.Render("Cancelled.")
	}
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(m.title))
	sb.WriteString("\n\n")
	sb.WriteString(m.form.View())
	sb.WriteString("\n")
	sb.WriteString(styleStatusBar.Render("esc abort  •  enter confirm field  •  ctrl+c quit"))
	return sb.String()
}

func (m FormModel) Done() bool    { return m.done }
func (m FormModel) Aborted() bool { return m.aborted }
func (m FormModel) Result() vm.VMConfig {
	return m.result
}

// --- validators ---

func validateInt(s string) error {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v <= 0 {
		return fmt.Errorf("must be a positive integer")
	}
	return nil
}

func validatePort(s string) error {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 1024 || v > 65535 {
		return fmt.Errorf("must be a port number 1024–65535")
	}
	return nil
}

func mustAtoi(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func defaultFormInt(value int, fallback int) string {
	if value <= 0 {
		value = fallback
	}
	return strconv.Itoa(value)
}
