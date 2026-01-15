package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

type View int

const (
	ViewMenu View = iota
	ViewContextList
	ViewNamespaceSelector
)

type Model struct {
	service         *application.Service
	currentView     View
	menuItems       []string
	menuCursor      int
	contexts        []application.ContextInfo
	contextCursor   int
	namespaces      []string
	namespaceCursor int
	width           int
	height          int
	statusMessage   string
	errorMessage    string
	quitting        bool
}

func NewModel() Model {
	repo := infrastructure.NewFileRepository()
	service := application.NewService(repo)

	return Model{
		service:     service,
		currentView: ViewMenu,
		menuItems: []string{
			"Switch Context",
			"Set Namespace",
			"List Contexts",
			"Current Info",
			"Exit",
		},
		menuCursor: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadContexts
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.errorMessage != "" && msg.String() != "q" && msg.String() != "ctrl+c" {
			m.errorMessage = ""
			return m, nil
		}

		switch {
		case key.Matches(msg, Keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, Keys.Back):
			if m.currentView != ViewMenu {
				m.currentView = ViewMenu
				m.statusMessage = ""
			}
			return m, nil
		}

		switch m.currentView {
		case ViewMenu:
			return m.updateMenu(msg)
		case ViewContextList:
			return m.updateContextList(msg)
		case ViewNamespaceSelector:
			return m.updateNamespaceSelector(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case contextsLoadedMsg:
		m.contexts = msg.contexts
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
		}
		return m, nil

	case namespacesLoadedMsg:
		m.namespaces = msg.namespaces
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
		}
		return m, nil

	case statusMsg:
		m.statusMessage = string(msg)
		return m, nil

	case errorMsg:
		m.errorMessage = string(msg)
		return m, nil
	}

	return m, nil
}

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.menuCursor < len(m.menuItems)-1 {
			m.menuCursor++
		}
	case key.Matches(msg, Keys.Select):
		return m.selectMenuItem()
	}
	return m, nil
}

func (m Model) selectMenuItem() (tea.Model, tea.Cmd) {
	switch m.menuItems[m.menuCursor] {
	case "Switch Context":
		m.currentView = ViewContextList
		m.contextCursor = 0
		return m, m.loadContexts
	case "Set Namespace":
		m.currentView = ViewNamespaceSelector
		m.namespaceCursor = 0
		return m, m.loadNamespaces
	case "List Contexts":
		m.currentView = ViewContextList
		m.contextCursor = 0
		return m, m.loadContexts
	case "Current Info":
		return m, m.showCurrentInfo
	case "Exit":
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) updateContextList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.contextCursor > 0 {
			m.contextCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.contextCursor < len(m.contexts)-1 {
			m.contextCursor++
		}
	case key.Matches(msg, Keys.Select):
		if len(m.contexts) > 0 {
			return m, m.switchContext(m.contexts[m.contextCursor].Name)
		}
	}
	return m, nil
}

func (m Model) updateNamespaceSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.namespaceCursor > 0 {
			m.namespaceCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.namespaceCursor < len(m.namespaces)-1 {
			m.namespaceCursor++
		}
	case key.Matches(msg, Keys.Select):
		if len(m.namespaces) > 0 {
			return m, m.setNamespace(m.namespaces[m.namespaceCursor])
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var content string

	switch m.currentView {
	case ViewMenu:
		content = m.viewMenu()
	case ViewContextList:
		content = m.viewContextList()
	case ViewNamespaceSelector:
		content = m.viewNamespaceSelector()
	}

	if m.errorMessage != "" {
		content += "\n\n" + ErrorStyle.Render(IconCross+" "+m.errorMessage)
	}

	if m.statusMessage != "" {
		content += "\n\n" + SuccessStyle.Render(IconCheck+" "+m.statusMessage)
	}

	content += "\n\n" + HelpStyle.Render("↑/k up • ↓/j down • enter select • esc back • q quit")

	return BoxStyle.Render(content)
}

func (m Model) viewMenu() string {
	var b strings.Builder

	title := TitleStyle.Render(IconMenu + " kubecfg")
	subtitle := SubtitleStyle.Render("Kubernetes Config Manager")
	b.WriteString(title + "\n" + subtitle + "\n\n")

	for i, item := range m.menuItems {
		if i == m.menuCursor {
			b.WriteString(SelectedItemStyle.Render(IconCurrent + " " + item))
		} else {
			b.WriteString(NormalItemStyle.Render(item))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) viewContextList() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render(IconContext+" Contexts") + "\n\n")

	if len(m.contexts) == 0 {
		b.WriteString(SubtitleStyle.Render("No contexts found"))
		return b.String()
	}

	for i, ctx := range m.contexts {
		line := m.formatContextLine(ctx)
		if i == m.contextCursor {
			b.WriteString(SelectedItemStyle.Render(IconCurrent + " " + line))
		} else {
			prefix := "  "
			if ctx.Current {
				prefix = CurrentMarkerStyle.Render(IconCheck + " ")
			}
			b.WriteString(NormalItemStyle.Render(prefix + line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) formatContextLine(ctx application.ContextInfo) string {
	name := ContextNameStyle.Render(ctx.Name)
	cluster := ClusterStyle.Render(ctx.Cluster)
	ns := ctx.Namespace
	if ns == "" {
		ns = "default"
	}
	namespace := NamespaceStyle.Render(ns)

	return fmt.Sprintf("%s (%s) [%s]", name, cluster, namespace)
}

func (m Model) viewNamespaceSelector() string {
	var b strings.Builder

	b.WriteString(HeaderStyle.Render(IconNamespace+" Namespaces") + "\n\n")

	if len(m.namespaces) == 0 {
		b.WriteString(SubtitleStyle.Render("No namespaces found"))
		return b.String()
	}

	for i, ns := range m.namespaces {
		if i == m.namespaceCursor {
			b.WriteString(SelectedItemStyle.Render(IconCurrent + " " + ns))
		} else {
			b.WriteString(NormalItemStyle.Render(ns))
		}
		b.WriteString("\n")
	}

	return b.String()
}

type contextsLoadedMsg struct {
	contexts []application.ContextInfo
	err      error
}

type namespacesLoadedMsg struct {
	namespaces []string
	err        error
}

type statusMsg string
type errorMsg string

func (m Model) loadContexts() tea.Msg {
	kubeconfigPath := config.GetKubeconfigPath()
	contexts, err := m.service.ListContexts(kubeconfigPath)
	return contextsLoadedMsg{contexts: contexts, err: err}
}

func (m Model) loadNamespaces() tea.Msg {
	kubeconfigPath := config.GetKubeconfigPath()
	k8sClient := infrastructure.NewKubernetesClient(kubeconfigPath)
	namespaces, err := k8sClient.ListNamespaces()
	return namespacesLoadedMsg{namespaces: namespaces, err: err}
}

func (m Model) switchContext(name string) tea.Cmd {
	return func() tea.Msg {
		kubeconfigPath := config.GetKubeconfigPath()
		if err := m.service.UseContext(kubeconfigPath, name, ""); err != nil {
			return errorMsg(err.Error())
		}
		return statusMsg(fmt.Sprintf("Switched to context '%s'", name))
	}
}

func (m Model) setNamespace(namespace string) tea.Cmd {
	return func() tea.Msg {
		kubeconfigPath := config.GetKubeconfigPath()
		if err := m.service.SetNamespace(kubeconfigPath, namespace); err != nil {
			return errorMsg(err.Error())
		}
		return statusMsg(fmt.Sprintf("Namespace set to '%s'", namespace))
	}
}

func (m Model) showCurrentInfo() tea.Msg {
	kubeconfigPath := config.GetKubeconfigPath()
	contexts, err := m.service.ListContexts(kubeconfigPath)
	if err != nil {
		return errorMsg(err.Error())
	}

	for _, ctx := range contexts {
		if ctx.Current {
			ns := ctx.Namespace
			if ns == "" {
				ns = "default"
			}
			info := fmt.Sprintf("Context: %s | Cluster: %s | Namespace: %s",
				ctx.Name, ctx.Cluster, ns)
			return statusMsg(info)
		}
	}
	return errorMsg("No current context set")
}

func Run() error {
	config.Init()
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func RunWithConfig(kubeconfigPath string) error {
	config.Init()
	config.SetKubeconfigPath(kubeconfigPath)
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
