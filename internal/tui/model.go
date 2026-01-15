package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	menuCursor      int
	contexts        []application.ContextInfo
	filteredCtx     []application.ContextInfo
	contextCursor   int
	namespaces      []string
	filteredNs      []string
	namespaceCursor int
	width           int
	height          int
	statusMessage   string
	errorMessage    string
	quitting        bool
	filterInput     textinput.Model
	filtering       bool
}

func NewModel() Model {
	repo := infrastructure.NewFileRepository()
	service := application.NewService(repo)

	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.CharLimit = 50
	ti.Width = 30

	return Model{
		service:     service,
		currentView: ViewMenu,
		menuCursor:  0,
		filterInput: ti,
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

		if m.filtering {
			return m.updateFiltering(msg)
		}

		switch {
		case key.Matches(msg, Keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, Keys.Back):
			if m.currentView != ViewMenu {
				m.currentView = ViewMenu
				m.statusMessage = ""
				m.resetFilter()
			}
			return m, nil

		case key.Matches(msg, Keys.Search):
			if m.currentView != ViewMenu {
				m.filtering = true
				m.filterInput.Focus()
				return m, textinput.Blink
			}
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
		m.filteredCtx = msg.contexts
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
		}
		return m, nil

	case namespacesLoadedMsg:
		m.namespaces = msg.namespaces
		m.filteredNs = msg.namespaces
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

func (m *Model) resetFilter() {
	m.filtering = false
	m.filterInput.Reset()
	m.filteredCtx = m.contexts
	m.filteredNs = m.namespaces
	m.contextCursor = 0
	m.namespaceCursor = 0
}

func (m Model) updateFiltering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.resetFilter()
		return m, nil
	case tea.KeyEnter:
		m.filtering = false
		m.filterInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)

	query := strings.ToLower(m.filterInput.Value())

	if m.currentView == ViewContextList {
		m.filteredCtx = filterContexts(m.contexts, query)
		m.contextCursor = 0
	} else if m.currentView == ViewNamespaceSelector {
		m.filteredNs = filterStrings(m.namespaces, query)
		m.namespaceCursor = 0
	}

	return m, cmd
}

func filterContexts(contexts []application.ContextInfo, query string) []application.ContextInfo {
	if query == "" {
		return contexts
	}
	var filtered []application.ContextInfo
	for _, ctx := range contexts {
		if strings.Contains(strings.ToLower(ctx.Name), query) ||
			strings.Contains(strings.ToLower(ctx.Cluster), query) ||
			strings.Contains(strings.ToLower(ctx.Namespace), query) {
			filtered = append(filtered, ctx)
		}
	}
	return filtered
}

func filterStrings(items []string, query string) []string {
	if query == "" {
		return items
	}
	var filtered []string
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.menuCursor < len(MenuItems)-1 {
			m.menuCursor++
		}
	case key.Matches(msg, Keys.Select):
		return m.selectMenuItem()
	}
	return m, nil
}

func (m Model) selectMenuItem() (tea.Model, tea.Cmd) {
	switch MenuItems[m.menuCursor].Label {
	case "Switch Context":
		m.currentView = ViewContextList
		m.contextCursor = 0
		m.resetFilter()
		return m, m.loadContexts
	case "Set Namespace":
		m.currentView = ViewNamespaceSelector
		m.namespaceCursor = 0
		m.resetFilter()
		return m, m.loadNamespaces
	case "List Contexts":
		m.currentView = ViewContextList
		m.contextCursor = 0
		m.resetFilter()
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
		if m.contextCursor < len(m.filteredCtx)-1 {
			m.contextCursor++
		}
	case key.Matches(msg, Keys.Select):
		if len(m.filteredCtx) > 0 {
			return m, m.switchContext(m.filteredCtx[m.contextCursor].Name)
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.filtering = true
			m.filterInput.Focus()
			m.filterInput.SetValue(string(msg.Runes))
			m.filterInput.SetCursor(len(msg.Runes))
			query := strings.ToLower(m.filterInput.Value())
			m.filteredCtx = filterContexts(m.contexts, query)
			m.contextCursor = 0
			return m, textinput.Blink
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
		if m.namespaceCursor < len(m.filteredNs)-1 {
			m.namespaceCursor++
		}
	case key.Matches(msg, Keys.Select):
		if len(m.filteredNs) > 0 {
			return m, m.setNamespace(m.filteredNs[m.namespaceCursor])
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.filtering = true
			m.filterInput.Focus()
			m.filterInput.SetValue(string(msg.Runes))
			m.filterInput.SetCursor(len(msg.Runes))
			query := strings.ToLower(m.filterInput.Value())
			m.filteredNs = filterStrings(m.namespaces, query)
			m.namespaceCursor = 0
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var content strings.Builder

	content.WriteString(m.renderHeader())
	content.WriteString("\n")

	switch m.currentView {
	case ViewMenu:
		content.WriteString(m.viewMenu())
	case ViewContextList:
		content.WriteString(m.viewContextList())
	case ViewNamespaceSelector:
		content.WriteString(m.viewNamespaceSelector())
	}

	if m.errorMessage != "" {
		content.WriteString("\n")
		content.WriteString(ErrorStyle.Render(" " + IconCross + " " + m.errorMessage))
	}

	if m.statusMessage != "" {
		content.WriteString("\n")
		content.WriteString(SuccessStyle.Render(" " + IconCheck + " " + m.statusMessage))
	}

	content.WriteString("\n\n")
	content.WriteString(m.renderHelp())

	mainContent := content.String()

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			mainContent,
		)
	}

	return mainContent
}

func (m Model) renderHeader() string {
	logo := LogoStyle.Render("⎈")
	name := AppNameStyle.Render("kubecfg")
	desc := DescStyle.Render("Kubernetes Config Manager")

	return fmt.Sprintf(" %s %s  %s", logo, name, desc)
}

func (m Model) renderHelp() string {
	var parts []string

	addKey := func(key, desc string) {
		parts = append(parts, HelpKeyStyle.Render(key)+HelpDescStyle.Render(" "+desc))
	}

	addKey("↑↓", "navigate")
	addKey("enter", "select")
	if m.currentView != ViewMenu {
		addKey("type", "filter")
		addKey("esc", "back")
	}
	addKey("q", "quit")

	return " " + strings.Join(parts, "  ")
}

func (m Model) viewMenu() string {
	var b strings.Builder

	b.WriteString("\n")

	for i, item := range MenuItems {
		var line string
		icon := DimItemStyle.Render(item.Icon)

		if i == m.menuCursor {
			cursor := SelectedItemStyle.Render(IconCurrent)
			label := SelectedItemStyle.Render(item.Label)
			line = fmt.Sprintf(" %s %s %s", cursor, icon, label)
		} else {
			label := NormalItemStyle.Render(item.Label)
			line = fmt.Sprintf("   %s %s", icon, label)
		}

		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m Model) viewContextList() string {
	var b strings.Builder

	title := HeaderStyle.Render(IconContext + " Contexts")
	b.WriteString("\n " + title)

	if m.filtering || m.filterInput.Value() != "" {
		filterStyle := FilterInputStyle.Render(" " + m.filterInput.View())
		b.WriteString("  " + filterStyle)
	}

	b.WriteString("\n\n")

	if len(m.filteredCtx) == 0 {
		if m.filterInput.Value() != "" {
			b.WriteString(" " + DimItemStyle.Render("No matches for '"+m.filterInput.Value()+"'") + "\n")
		} else {
			b.WriteString(" " + DimItemStyle.Render("No contexts found") + "\n")
		}
		return b.String()
	}

	maxVisible := 10
	start := 0
	if m.contextCursor >= maxVisible {
		start = m.contextCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.filteredCtx) {
		end = len(m.filteredCtx)
	}

	for i := start; i < end; i++ {
		ctx := m.filteredCtx[i]
		line := m.formatContextLine(ctx)

		if i == m.contextCursor {
			cursor := SelectedItemStyle.Render(IconCurrent)
			b.WriteString(fmt.Sprintf(" %s %s\n", cursor, line))
		} else if ctx.Current {
			marker := CurrentMarkerStyle.Render(IconCheck)
			b.WriteString(fmt.Sprintf(" %s %s\n", marker, line))
		} else {
			b.WriteString(fmt.Sprintf("   %s\n", line))
		}
	}

	if len(m.filteredCtx) > maxVisible {
		info := DimItemStyle.Render(fmt.Sprintf(" [%d/%d]", m.contextCursor+1, len(m.filteredCtx)))
		b.WriteString("\n" + info)
	}

	return b.String()
}

func (m Model) formatContextLine(ctx application.ContextInfo) string {
	name := ContextNameStyle.Render(ctx.Name)

	ns := ctx.Namespace
	if ns == "" {
		ns = "default"
	}

	details := DimItemStyle.Render(fmt.Sprintf("(%s)", ns))

	return fmt.Sprintf("%s %s", name, details)
}

func (m Model) viewNamespaceSelector() string {
	var b strings.Builder

	title := HeaderStyle.Render(IconNamespace + " Namespaces")
	b.WriteString("\n " + title)

	if m.filtering || m.filterInput.Value() != "" {
		filterStyle := FilterInputStyle.Render(" " + m.filterInput.View())
		b.WriteString("  " + filterStyle)
	}

	b.WriteString("\n\n")

	if len(m.filteredNs) == 0 {
		if m.filterInput.Value() != "" {
			b.WriteString(" " + DimItemStyle.Render("No matches for '"+m.filterInput.Value()+"'") + "\n")
		} else {
			b.WriteString(" " + DimItemStyle.Render("No namespaces found") + "\n")
		}
		return b.String()
	}

	maxVisible := 10
	start := 0
	if m.namespaceCursor >= maxVisible {
		start = m.namespaceCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.filteredNs) {
		end = len(m.filteredNs)
	}

	for i := start; i < end; i++ {
		ns := m.filteredNs[i]

		if i == m.namespaceCursor {
			cursor := SelectedItemStyle.Render(IconCurrent)
			label := SelectedItemStyle.Render(ns)
			b.WriteString(fmt.Sprintf(" %s %s\n", cursor, label))
		} else {
			label := NormalItemStyle.Render(ns)
			b.WriteString(fmt.Sprintf("   %s\n", label))
		}
	}

	if len(m.filteredNs) > maxVisible {
		info := DimItemStyle.Render(fmt.Sprintf(" [%d/%d]", m.namespaceCursor+1, len(m.filteredNs)))
		b.WriteString("\n" + info)
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
		return statusMsg(fmt.Sprintf("Switched to '%s'", name))
	}
}

func (m Model) setNamespace(namespace string) tea.Cmd {
	return func() tea.Msg {
		kubeconfigPath := config.GetKubeconfigPath()
		if err := m.service.SetNamespace(kubeconfigPath, namespace); err != nil {
			return errorMsg(err.Error())
		}
		return statusMsg(fmt.Sprintf("Namespace: %s", namespace))
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
			return statusMsg(fmt.Sprintf("%s → %s", ctx.Name, ns))
		}
	}
	return errorMsg("No current context")
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
