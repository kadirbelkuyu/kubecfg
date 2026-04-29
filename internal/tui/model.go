package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	appgroupservice "github.com/kadirbelkuyu/kubecfg/internal/application/groupservice"
	"github.com/kadirbelkuyu/kubecfg/internal/application/healthservice"
	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
	healthdomain "github.com/kadirbelkuyu/kubecfg/internal/domain/health"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/groupstore"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/healthcheck"
)

type View int

const (
	ViewMenu View = iota
	ViewContextList
	ViewNamespaceSelector
	ViewAddContext
	ViewRenameContext
	ViewRemoveConfirm
	ViewGuard
	ViewPolicy
	ViewGroup
	ViewKubeconfigSource
	ViewConfirmModal
)

type InputMode int

const (
	InputNone InputMode = iota
	InputFilePath
	InputContextName
	InputNewName
)

type Model struct {
	service         *application.Service
	groupService    *appgroupservice.Service
	guardService    *application.GuardService
	policyService   *application.PolicyService
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
	inputMode       InputMode
	textInput       textinput.Model
	selectedContext string
	guardStatus     *application.GuardStatus
	guardCursor     int
	guardTTLIndex   int
	guardTTLOptions []time.Duration
	// policy view state
	policies        []domain.Policy
	policyCursor    int
	policyUserFlags map[string]bool // name → true if user-defined
	groups          []domaingroup.Group
	groupCursor     int
	sources         []application.KubeconfigSourceInfo
	sourceCursor    int
	// confirmation modal state
	confirmationStore domain.ConfirmationStore
	pendingConfirm    *domain.PendingConfirmation
	previousView      View
	healthService     *healthservice.Service
	healthResults     map[string]healthdomain.Result
	healthChecking    bool
	healthPending     int
	healthSpinner     int
	healthLoaded      bool
}

func NewModel(healthSvc *healthservice.Service) (Model, error) {
	repo := infrastructure.NewFileRepository()
	service := application.NewService(
		repo,
		application.WithPreviousContextStore(infrastructure.NewPreviousContextStore(config.GetLastContextPath())),
	)
	runtime, err := infrastructure.NewGuardProcessRuntime("", config.GetGuardSessionPath())
	if err != nil {
		return Model{}, fmt.Errorf("create guard runtime: %w", err)
	}
	auditStore := infrastructure.NewAuditFileStore(config.GetAuditPath())
	auditService := application.NewAuditService(auditStore, config.IsAuditEnabled())
	policySvc := application.NewPolicyService(config.GetProfiles())
	guardWriter := infrastructure.NewGuardKubeconfigWriter()
	sessionStore := infrastructure.NewSessionFileStore(config.GetGuardSessionPath())
	sessionService := application.NewSessionService(sessionStore, runtime, guardWriter, auditService)
	guardService := application.NewGuardService(
		repo,
		sessionService,
		guardWriter,
		runtime,
		auditService,
		filepath.Join(config.GetGuardStateDir(), "guard"),
		config.GetGuardDefaultTTL(),
		application.WithGuardPolicyResolver(policySvc),
	)

	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.CharLimit = 50
	ti.Width = 30

	input := textinput.New()
	input.CharLimit = 100
	input.Width = 40

	groupSvc := appgroupservice.NewService(
		groupstore.NewFileStore(config.GetGroupsPath()),
		repo,
		config.GetKubeconfigPath(),
		appgroupservice.WithPolicyResolver(policySvc),
	)
	confirmStore := infrastructure.NewFileConfirmationStore(config.GetConfirmationsDir())
	if healthSvc == nil {
		healthSvc = healthservice.New(
			healthcheck.New(config.GetKubeconfigPath()),
			healthcheck.NewCache(),
			repo,
			config.GetKubeconfigPath(),
		)
	}

	return Model{
		service:           service,
		groupService:      groupSvc,
		guardService:      guardService,
		policyService:     policySvc,
		confirmationStore: confirmStore,
		currentView:       ViewMenu,
		menuCursor:        0,
		filterInput:       ti,
		textInput:         input,
		guardTTLOptions:   []time.Duration{15 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour},
		guardTTLIndex:     1,
		policyUserFlags:   make(map[string]bool),
		healthService:     healthSvc,
		healthResults:     make(map[string]healthdomain.Result),
	}, nil
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadContexts, m.loadGuardStatus, tickConfirmPoll())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.errorMessage != "" && msg.String() != "q" && msg.String() != "ctrl+c" {
			m.errorMessage = ""
			return m, nil
		}

		if m.inputMode != InputNone {
			return m.updateTextInput(msg)
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
				m.inputMode = InputNone
			}
			return m, nil

		case key.Matches(msg, Keys.Search):
			if m.currentView == ViewContextList || m.currentView == ViewNamespaceSelector {
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
		case ViewRemoveConfirm:
			return m.updateRemoveConfirm(msg)
		case ViewGuard:
			return m.updateGuard(msg)
		case ViewPolicy:
			return m.updatePolicy(msg)
		case ViewGroup:
			return m.updateGroup(msg)
		case ViewKubeconfigSource:
			return m.updateKubeconfigSource(msg)
		case ViewConfirmModal:
			return m.updateConfirmModal(msg)
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
			return m, nil
		}
		if !m.healthLoaded {
			return m, m.startHealthChecksFor(contextNames(msg.contexts), false)
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

	case contextAddedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
		} else {
			m.statusMessage = fmt.Sprintf("Added context '%s'", msg.name)
			m.currentView = ViewMenu
			m.inputMode = InputNone
		}
		return m, m.loadContexts

	case contextRenamedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
		} else {
			m.statusMessage = fmt.Sprintf("Renamed to '%s'", msg.newName)
			m.currentView = ViewMenu
			m.inputMode = InputNone
		}
		return m, m.loadContexts

	case contextRemovedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
		} else {
			m.statusMessage = fmt.Sprintf("Removed context '%s'", msg.name)
			m.currentView = ViewMenu
		}
		return m, m.loadContexts

	case guardStatusLoadedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.guardStatus = msg.status
		if msg.status != nil && msg.status.Active {
			return m, tickGuardStatus()
		}
		return m, nil

	case guardStartedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Guard started: export KUBECONFIG=%s", msg.session.GeneratedKubeconfigPath)
		return m, m.loadGuardStatus

	case guardStoppedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.statusMessage = "Guard stopped"
		return m, m.loadGuardStatus

	case guardTickMsg:
		if m.guardStatus != nil && m.guardStatus.Active {
			return m, m.loadGuardStatus
		}
		return m, nil

	case confirmPollMsg:
		if m.confirmationStore != nil && m.currentView != ViewConfirmModal {
			store := m.confirmationStore
			return m, func() tea.Msg {
				pending, err := store.ListPending()
				if err != nil || len(pending) == 0 {
					return tickConfirmPoll()()
				}
				return confirmPendingMsg{pending: pending[0]}
			}
		}
		return m, tickConfirmPoll()

	case confirmPendingMsg:
		m.pendingConfirm = msg.pending
		m.previousView = m.currentView
		m.currentView = ViewConfirmModal
		return m, nil

	case healthRefreshRequestMsg:
		m.statusMessage = "Refreshing health status..."
		return m, m.startHealthChecksFor(contextNames(m.filteredCtx), true)

	case healthCheckProgressMsg:
		m.healthResults[msg.result.ContextName] = msg.result
		if m.healthPending > 0 {
			m.healthPending--
		}
		if m.healthPending == 0 {
			results := make([]healthdomain.Result, 0, len(m.healthResults))
			for _, contextInfo := range m.contexts {
				if result, ok := m.healthResults[contextInfo.Name]; ok {
					results = append(results, result)
				}
			}
			return m, func() tea.Msg {
				return healthCheckCompleteMsg{results: results}
			}
		}
		return m, nil

	case healthCheckCompleteMsg:
		for _, result := range msg.results {
			m.healthResults[result.ContextName] = result
		}
		m.healthChecking = false
		m.healthPending = 0
		m.healthLoaded = true
		if m.statusMessage == "Refreshing health status..." {
			m.statusMessage = "Health status refreshed"
		}
		return m, nil

	case healthSpinnerTickMsg:
		if !m.healthChecking {
			return m, nil
		}
		m.healthSpinner = (m.healthSpinner + 1) % len(healthSpinnerFrames)
		return m, tickHealthSpinner()

	case policiesLoadedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.policies = msg.policies
		m.policyUserFlags = msg.policyUserFlags
		return m, nil

	case groupsLoadedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.groups = msg.groups
		if m.groupCursor >= len(m.groups) {
			m.groupCursor = 0
		}
		return m, nil

	case kubeconfigSourcesLoadedMsg:
		m.sources = msg.sources
		if m.sourceCursor >= len(m.sources) {
			m.sourceCursor = 0
		}
		return m, nil

	case kubeconfigSourceActivatedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.rebuildKubeconfigScopedServices(msg.path)
		m.statusMessage = fmt.Sprintf("Kubeconfig source: %s", msg.path)
		m.currentView = ViewContextList
		m.contextCursor = 0
		m.resetFilter()
		return m, tea.Batch(m.loadContexts, m.loadKubeconfigSources)

	case groupUsedMsg:
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.statusMessage = msg.message
		return m, tea.Batch(m.loadContexts, m.loadGuardStatus)

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			switch m.currentView {
			case ViewMenu:
				return m.selectMenuItem()
			case ViewContextList:
				if len(m.filteredCtx) > 0 {
					menuLabel := MenuItems[m.menuCursor].Label
					ctx := m.filteredCtx[m.contextCursor]
					switch menuLabel {
					case "Rename Context":
						m.currentView = ViewRenameContext
						m.inputMode = InputNewName
						m.selectedContext = ctx.Name
						m.textInput.Reset()
						m.textInput.Placeholder = "new name..."
						m.textInput.SetValue(ctx.Name)
						m.textInput.Focus()
						return m, textinput.Blink
					case "Remove Context":
						m.currentView = ViewRemoveConfirm
						m.selectedContext = ctx.Name
						return m, nil
					default:
						return m, m.switchContext(ctx.Name)
					}
				}
			case ViewNamespaceSelector:
				if len(m.filteredNs) > 0 {
					return m, m.setNamespace(m.filteredNs[m.namespaceCursor])
				}
			case ViewGuard:
				return m, m.selectGuardAction()
			case ViewGroup:
				if len(m.groups) > 0 {
					return m, m.useGroup(m.groups[m.groupCursor])
				}
			case ViewKubeconfigSource:
				if len(m.sources) > 0 {
					return m, m.activateKubeconfigSource(m.sources[m.sourceCursor].Path)
				}
			}
		}
		if msg.Button == tea.MouseButtonWheelUp {
			switch m.currentView {
			case ViewMenu:
				if m.menuCursor > 0 {
					m.menuCursor--
				}
			case ViewContextList:
				if m.contextCursor > 0 {
					m.contextCursor--
				}
			case ViewNamespaceSelector:
				if m.namespaceCursor > 0 {
					m.namespaceCursor--
				}
			case ViewGroup:
				if m.groupCursor > 0 {
					m.groupCursor--
				}
			case ViewKubeconfigSource:
				if m.sourceCursor > 0 {
					m.sourceCursor--
				}
			}
			return m, nil
		}
		if msg.Button == tea.MouseButtonWheelDown {
			switch m.currentView {
			case ViewMenu:
				if m.menuCursor < len(MenuItems)-1 {
					m.menuCursor++
				}
			case ViewContextList:
				if m.contextCursor < len(m.filteredCtx)-1 {
					m.contextCursor++
				}
			case ViewNamespaceSelector:
				if m.namespaceCursor < len(m.filteredNs)-1 {
					m.namespaceCursor++
				}
			case ViewGroup:
				if m.groupCursor < len(m.groups)-1 {
					m.groupCursor++
				}
			case ViewKubeconfigSource:
				if m.sourceCursor < len(m.sources)-1 {
					m.sourceCursor++
				}
			}
			return m, nil
		}
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

func (m Model) updateTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.inputMode = InputNone
		m.currentView = ViewMenu
		m.textInput.Reset()
		return m, nil
	case tea.KeyEnter:
		value := m.textInput.Value()
		if value == "" {
			return m, nil
		}

		switch m.inputMode {
		case InputFilePath:
			m.inputMode = InputContextName
			m.textInput.Reset()
			m.textInput.Placeholder = "context name..."
			m.textInput.Focus()
			m.selectedContext = value
			return m, textinput.Blink
		case InputContextName:
			return m, m.addContext(m.selectedContext, value)
		case InputNewName:
			return m, m.renameContext(m.selectedContext, value)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
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
	case tea.KeyUp:
		if m.currentView == ViewContextList && m.contextCursor > 0 {
			m.contextCursor--
		} else if m.currentView == ViewNamespaceSelector && m.namespaceCursor > 0 {
			m.namespaceCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.currentView == ViewContextList && m.contextCursor < len(m.filteredCtx)-1 {
			m.contextCursor++
		} else if m.currentView == ViewNamespaceSelector && m.namespaceCursor < len(m.filteredNs)-1 {
			m.namespaceCursor++
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)

	query := strings.ToLower(m.filterInput.Value())

	switch m.currentView {
	case ViewContextList:
		m.filteredCtx = filterContexts(m.contexts, query)
		m.contextCursor = 0
	case ViewNamespaceSelector:
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
	case "Kubeconfig Sources":
		m.currentView = ViewKubeconfigSource
		m.sourceCursor = 0
		return m, m.loadKubeconfigSources
	case "Context Groups":
		m.currentView = ViewGroup
		m.groupCursor = 0
		return m, m.loadGroups
	case "Guard":
		m.currentView = ViewGuard
		m.guardCursor = 0
		return m, m.loadGuardStatus
	case "Policies":
		m.currentView = ViewPolicy
		m.policyCursor = 0
		return m, m.loadPolicies
	case "Add Context":
		m.currentView = ViewAddContext
		m.inputMode = InputFilePath
		m.textInput.Reset()
		m.textInput.Placeholder = "path to kubeconfig file..."
		m.textInput.Focus()
		return m, textinput.Blink
	case "Rename Context":
		m.currentView = ViewContextList
		m.contextCursor = 0
		m.resetFilter()
		return m, m.loadContexts
	case "Remove Context":
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
	menuLabel := MenuItems[m.menuCursor].Label

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
			ctx := m.filteredCtx[m.contextCursor]
			switch menuLabel {
			case "Rename Context":
				m.currentView = ViewRenameContext
				m.inputMode = InputNewName
				m.selectedContext = ctx.Name
				m.textInput.Reset()
				m.textInput.Placeholder = "new name..."
				m.textInput.SetValue(ctx.Name)
				m.textInput.Focus()
				return m, textinput.Blink
			case "Remove Context":
				m.currentView = ViewRemoveConfirm
				m.selectedContext = ctx.Name
				return m, nil
			default:
				return m, m.switchContext(ctx.Name)
			}
		}
	case key.Matches(msg, Keys.Refresh):
		return m, func() tea.Msg {
			return healthRefreshRequestMsg{}
		}
	default:
		if len(m.filteredCtx) > 0 {
			ctx := m.filteredCtx[m.contextCursor]
			switch msg.String() {
			case "ctrl+r":
				m.currentView = ViewRenameContext
				m.inputMode = InputNewName
				m.selectedContext = ctx.Name
				m.textInput.Reset()
				m.textInput.Placeholder = "new name..."
				m.textInput.SetValue(ctx.Name)
				m.textInput.Focus()
				return m, textinput.Blink
			case "ctrl+d":
				m.currentView = ViewRemoveConfirm
				m.selectedContext = ctx.Name
				return m, nil
			}
		}
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

func (m Model) updateRemoveConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, m.removeContext(m.selectedContext)
	case "n", "N", "esc":
		m.currentView = ViewMenu
		m.selectedContext = ""
		return m, nil
	}
	return m, nil
}

func (m Model) updateGroup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.groupCursor > 0 {
			m.groupCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.groupCursor < len(m.groups)-1 {
			m.groupCursor++
		}
	case key.Matches(msg, Keys.Select):
		if len(m.groups) > 0 {
			return m, m.useGroup(m.groups[m.groupCursor])
		}
	case key.Matches(msg, Keys.Refresh):
		return m, m.loadGroups
	}
	return m, nil
}

func (m Model) updateKubeconfigSource(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.sourceCursor > 0 {
			m.sourceCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.sourceCursor < len(m.sources)-1 {
			m.sourceCursor++
		}
	case key.Matches(msg, Keys.Select):
		if len(m.sources) > 0 {
			return m, m.activateKubeconfigSource(m.sources[m.sourceCursor].Path)
		}
	case key.Matches(msg, Keys.Refresh):
		return m, m.loadKubeconfigSources
	}
	return m, nil
}

func (m Model) updateGuard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := m.guardActions()

	switch {
	case key.Matches(msg, Keys.Up):
		if m.guardCursor > 0 {
			m.guardCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.guardCursor < len(actions)-1 {
			m.guardCursor++
		}
	case key.Matches(msg, Keys.Select):
		return m, m.selectGuardAction()
	case key.Matches(msg, Keys.Refresh):
		return m, m.loadGuardStatus
	default:
		switch msg.String() {
		case "[":
			if m.guardTTLIndex > 0 {
				m.guardTTLIndex--
			}
		case "]":
			if m.guardTTLIndex < len(m.guardTTLOptions)-1 {
				m.guardTTLIndex++
			}
		}
	}

	return m, nil
}

func (m Model) updatePolicy(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.policyCursor > 0 {
			m.policyCursor--
		}
	case key.Matches(msg, Keys.Down):
		if m.policyCursor < len(m.policies)-1 {
			m.policyCursor++
		}
	case key.Matches(msg, Keys.Refresh):
		return m, m.loadPolicies
	}
	return m, nil
}

func (m Model) updateConfirmModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pendingConfirm == nil {
		m.currentView = m.previousView
		return m, tickConfirmPoll()
	}

	switch msg.String() {
	case "y", "Y":
		if m.confirmationStore != nil {
			_ = m.confirmationStore.Decide(m.pendingConfirm.ID, domain.ConfirmDecisionApproved)
		}
		m.pendingConfirm = nil
		m.currentView = m.previousView
		return m, tickConfirmPoll()
	case "n", "N", "esc":
		if m.confirmationStore != nil {
			_ = m.confirmationStore.Decide(m.pendingConfirm.ID, domain.ConfirmDecisionDenied)
		}
		m.pendingConfirm = nil
		m.currentView = m.previousView
		return m, tickConfirmPoll()
	}
	return m, nil
}

func (m Model) viewConfirmModal() string {
	if m.pendingConfirm == nil {
		return ""
	}
	p := m.pendingConfirm
	content := strings.Join([]string{
		ErrorStyle.Render("⚠  Destructive operation detected"),
		"",
		fmt.Sprintf("  Method:    %s", SelectedItemStyle.Render(p.Method)),
		fmt.Sprintf("  Resource:  %s", ContextNameStyle.Render(p.Resource)),
		fmt.Sprintf("  Namespace: %s", NamespaceStyle.Render(p.Namespace)),
		"",
		DimItemStyle.Render("Allow? [y] Yes  [n/esc] No  (30s timeout → denied)"),
	}, "\n")
	return ConfirmModalStyle.Render(content)
}

func (m Model) viewPolicies() string {
	var b strings.Builder

	b.WriteString(viewPolicyList(m.policies, m.policyUserFlags, m.policyCursor))

	if len(m.policies) > 0 && m.policyCursor < len(m.policies) {
		p := m.policies[m.policyCursor]
		isUser := m.policyUserFlags[p.Name]
		b.WriteString("\n")
		b.WriteString(viewPolicyDetail(&p, isUser))
	}

	return b.String()
}

func (m Model) viewGroups() string {
	var b strings.Builder

	b.WriteString("\n " + HeaderStyle.Render(IconGroup+" Context Groups") + "\n\n")
	if len(m.groups) == 0 {
		b.WriteString(" " + DimItemStyle.Render("No groups found") + "\n")
		return b.String()
	}

	maxVisible := 10
	start := 0
	if m.groupCursor >= maxVisible {
		start = m.groupCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.groups) {
		end = len(m.groups)
	}

	for i := start; i < end; i++ {
		g := m.groups[i]
		line := formatGroupLine(g)
		if i == m.groupCursor {
			cursor := SelectedItemStyle.Render(IconCurrent)
			_, _ = fmt.Fprintf(&b, " %s %s\n", cursor, SelectedItemStyle.Render(line))
			continue
		}
		_, _ = fmt.Fprintf(&b, "   %s\n", NormalItemStyle.Render(line))
	}

	if len(m.groups) > maxVisible {
		info := DimItemStyle.Render(fmt.Sprintf(" [%d/%d]", m.groupCursor+1, len(m.groups)))
		b.WriteString("\n" + info)
	}

	return b.String()
}

func formatGroupLine(g domaingroup.Group) string {
	parts := []string{
		g.Name,
		fmt.Sprintf("[%d]", len(g.Contexts)),
	}
	if g.Policy != "" {
		parts = append(parts, "policy:"+g.Policy)
	}
	if g.Description != "" {
		parts = append(parts, "- "+g.Description)
	}
	return strings.Join(parts, " ")
}

func (m Model) viewKubeconfigSources() string {
	var b strings.Builder

	b.WriteString("\n " + HeaderStyle.Render(IconSource+" Kubeconfig Sources") + "\n")
	b.WriteString(" " + DimItemStyle.Render("Active: "+config.GetKubeconfigPath()) + "\n\n")
	if len(m.sources) == 0 {
		b.WriteString(" " + DimItemStyle.Render("No kubeconfig sources found") + "\n")
		return b.String()
	}

	maxVisible := 10
	start := 0
	if m.sourceCursor >= maxVisible {
		start = m.sourceCursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.sources) {
		end = len(m.sources)
	}

	for i := start; i < end; i++ {
		source := m.sources[i]
		line := formatKubeconfigSourceLine(source)
		switch {
		case i == m.sourceCursor:
			cursor := SelectedItemStyle.Render(IconCurrent)
			_, _ = fmt.Fprintf(&b, " %s %s\n", cursor, SelectedItemStyle.Render(line))
		case source.Active:
			marker := CurrentMarkerStyle.Render(IconCheck)
			_, _ = fmt.Fprintf(&b, " %s %s\n", marker, NormalItemStyle.Render(line))
		default:
			_, _ = fmt.Fprintf(&b, "   %s\n", NormalItemStyle.Render(line))
		}
	}

	if len(m.sources) > maxVisible {
		info := DimItemStyle.Render(fmt.Sprintf(" [%d/%d]", m.sourceCursor+1, len(m.sources)))
		b.WriteString("\n" + info)
	}

	return b.String()
}

func formatKubeconfigSourceLine(source application.KubeconfigSourceInfo) string {
	status := fmt.Sprintf("%d contexts", source.ContextCount)
	if source.CurrentContext != "" {
		status += " current:" + source.CurrentContext
	}
	if source.Error != "" {
		status = "error:" + source.Error
	}
	return fmt.Sprintf("%s [%s]", source.Path, status)
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
	if banner := m.renderGuardBanner(); banner != "" {
		content.WriteString(banner)
		content.WriteString("\n\n")
	}

	switch m.currentView {
	case ViewMenu:
		content.WriteString(m.viewMenu())
	case ViewContextList:
		content.WriteString(m.viewContextList())
	case ViewNamespaceSelector:
		content.WriteString(m.viewNamespaceSelector())
	case ViewAddContext:
		content.WriteString(m.viewAddContext())
	case ViewRenameContext:
		content.WriteString(m.viewRenameContext())
	case ViewRemoveConfirm:
		content.WriteString(m.viewRemoveConfirm())
	case ViewGuard:
		content.WriteString(m.viewGuard())
	case ViewPolicy:
		content.WriteString(m.viewPolicies())
	case ViewGroup:
		content.WriteString(m.viewGroups())
	case ViewKubeconfigSource:
		content.WriteString(m.viewKubeconfigSources())
	case ViewConfirmModal:
		content.WriteString(m.viewConfirmModal())
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
	lines := []string{
		"██╗  ██╗██╗   ██╗██████╗ ███████╗ ██████╗███████╗ ██████╗ ",
		"██║ ██╔╝██║   ██║██╔══██╗██╔════╝██╔════╝██╔════╝██╔════╝ ",
		"█████╔╝ ██║   ██║██████╔╝█████╗  ██║     █████╗  ██║  ███╗",
		"██╔═██╗ ██║   ██║██╔══██╗██╔══╝  ██║     ██╔══╝  ██║   ██║",
		"██║  ██╗╚██████╔╝██████╔╝███████╗╚██████╗██║     ╚██████╔╝",
		"╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝ ╚═════╝╚═╝      ╚═════╝ ",
	}

	gradientColors := []lipgloss.Color{
		"#FFFFFF", "#DDDDDD", "#BBBBBB", "#999999", "#777777", "#555555",
	}

	renderedLines := make([]string, 0, len(lines))
	for i, line := range lines {
		style := lipgloss.NewStyle().Foreground(gradientColors[i]).Bold(true)
		renderedLines = append(renderedLines, style.Render(line))
	}

	return strings.Join(renderedLines, "\n")
}

func (m Model) renderHelp() string {
	var parts []string

	addKey := func(key, desc string) {
		parts = append(parts, HelpKeyStyle.Render(key)+HelpDescStyle.Render(" "+desc))
	}

	addKey("↑↓", "navigate")

	switch m.currentView {
	case ViewMenu:
		addKey("enter", "select")
	case ViewContextList:
		addKey("enter", "select")
		addKey("ctrl+r", "rename")
		addKey("ctrl+d", "delete")
		addKey("type", "filter")
		addKey("esc", "back")
	case ViewNamespaceSelector:
		addKey("enter", "select")
		addKey("type", "filter")
		addKey("esc", "back")
	case ViewAddContext, ViewRenameContext:
		addKey("enter", "confirm")
		addKey("esc", "cancel")
	case ViewRemoveConfirm:
		addKey("y/n", "confirm")
		addKey("esc", "cancel")
	case ViewGuard:
		addKey("enter", "select")
		addKey("r", "refresh")
		addKey("[ ]", "ttl")
		addKey("esc", "back")
	case ViewPolicy:
		addKey("r", "refresh")
		addKey("esc", "back")
	case ViewGroup:
		addKey("enter", "use")
		addKey("r", "refresh")
		addKey("esc", "back")
	case ViewKubeconfigSource:
		addKey("enter", "use")
		addKey("r", "refresh")
		addKey("esc", "back")
	case ViewConfirmModal:
		addKey("y", "approve")
		addKey("n/esc", "deny")
	}

	addKey("q", "quit")

	return " " + strings.Join(parts, "  ")
}

func (m Model) renderGuardBanner() string {
	if m.guardStatus == nil || !m.guardStatus.Active || m.guardStatus.Session == nil {
		return ""
	}

	session := m.guardStatus.Session
	profile := session.PolicyName

	label := fmt.Sprintf("[GUARD: %s]", strings.ToUpper(profile))
	if profile == "" {
		label = "[GUARD: READONLY]"
	}

	parts := []string{
		label,
		session.TargetContext,
		session.NamespaceDisplay(),
		formatGuardDuration(m.guardStatus.Remaining),
	}

	if m.width == 0 || m.width > 110 {
		parts = append(parts, session.ProxyListenAddress)
	}

	text := strings.Join(parts, " | ")

	switch profile {
	case domain.PolicyProfileProd:
		return ProfileProdStyle.Render(text)
	case domain.PolicyProfileStaging:
		return ProfileStagingStyle.Render(text)
	case domain.PolicyProfileDebug:
		return ProfileDebugStyle.Render(text)
	default:
		return GuardBannerStyle.Render(text)
	}
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

	menuLabel := MenuItems[m.menuCursor].Label
	var title string
	switch menuLabel {
	case "Rename Context":
		title = HeaderStyle.Render(IconRename + " Select context to rename")
	case "Remove Context":
		title = HeaderStyle.Render(IconRemove + " Select context to remove")
	default:
		title = HeaderStyle.Render(IconContext + " Contexts")
	}

	b.WriteString("\n " + title)
	b.WriteString("\n " + DimItemStyle.Render("Source: "+config.GetKubeconfigPath()))

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

		switch {
		case i == m.contextCursor:
			cursor := SelectedItemStyle.Render(IconCurrent)
			_, _ = fmt.Fprintf(&b, " %s %s\n", cursor, line)
		case ctx.Current:
			marker := CurrentMarkerStyle.Render(IconCheck)
			_, _ = fmt.Fprintf(&b, " %s %s\n", marker, line)
		default:
			_, _ = fmt.Fprintf(&b, "   %s\n", line)
		}
	}

	if len(m.filteredCtx) > maxVisible {
		info := DimItemStyle.Render(fmt.Sprintf(" [%d/%d]", m.contextCursor+1, len(m.filteredCtx)))
		b.WriteString("\n" + info)
	}

	return b.String()
}

func (m Model) formatContextLine(ctx application.ContextInfo) string {
	indicator, detail := m.healthIndicator(ctx.Name)
	name := ContextNameStyle.Render(ctx.Name)

	ns := ctx.Namespace
	if ns == "" {
		ns = "default"
	}

	namespace := DimItemStyle.Render("[" + ns + "]")
	if detail == "" {
		return fmt.Sprintf("%s %s %s", indicator, name, namespace)
	}

	return fmt.Sprintf("%s %s %s %s", indicator, name, detail, namespace)
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
			_, _ = fmt.Fprintf(&b, " %s %s\n", cursor, label)
		} else {
			label := NormalItemStyle.Render(ns)
			_, _ = fmt.Fprintf(&b, "   %s\n", label)
		}
	}

	if len(m.filteredNs) > maxVisible {
		info := DimItemStyle.Render(fmt.Sprintf(" [%d/%d]", m.namespaceCursor+1, len(m.filteredNs)))
		b.WriteString("\n" + info)
	}

	return b.String()
}

func (m Model) viewAddContext() string {
	var b strings.Builder

	title := HeaderStyle.Render(IconAdd + " Add Context")
	b.WriteString("\n " + title + "\n\n")

	switch m.inputMode {
	case InputFilePath:
		b.WriteString(" " + DimItemStyle.Render("Kubeconfig file path:") + "\n")
		b.WriteString(" " + m.textInput.View() + "\n")
	case InputContextName:
		b.WriteString(" " + DimItemStyle.Render("File: "+m.selectedContext) + "\n\n")
		b.WriteString(" " + DimItemStyle.Render("Context name:") + "\n")
		b.WriteString(" " + m.textInput.View() + "\n")
	}

	return b.String()
}

func (m Model) viewRenameContext() string {
	var b strings.Builder

	title := HeaderStyle.Render(IconRename + " Rename Context")
	b.WriteString("\n " + title + "\n\n")

	b.WriteString(" " + DimItemStyle.Render("Current: "+m.selectedContext) + "\n\n")
	b.WriteString(" " + DimItemStyle.Render("New name:") + "\n")
	b.WriteString(" " + m.textInput.View() + "\n")

	return b.String()
}

func (m Model) viewRemoveConfirm() string {
	var b strings.Builder

	title := HeaderStyle.Render(IconRemove + " Remove Context")
	b.WriteString("\n " + title + "\n\n")

	b.WriteString(" " + ErrorStyle.Render("Are you sure you want to remove?") + "\n\n")
	b.WriteString(" " + ContextNameStyle.Render(m.selectedContext) + "\n\n")
	b.WriteString(" " + DimItemStyle.Render("Press [y] to confirm, [n] or [esc] to cancel") + "\n")

	return b.String()
}

func (m Model) viewGuard() string {
	var b strings.Builder

	title := HeaderStyle.Render(IconGuard + " Guard")
	b.WriteString("\n " + title + "\n\n")

	if m.guardStatus == nil || m.guardStatus.Session == nil {
		b.WriteString(GuardPanelStyle.Render(strings.Join([]string{
			"Status: inactive",
			"Detail: no active guard session",
			"Readonly session: not started",
			"TTL preset: " + m.selectedGuardTTL().String(),
			"Press enter on Start Readonly Guard to create a temporary guarded kubeconfig",
		}, "\n")))
	} else {
		session := m.guardStatus.Session
		healthStyle := GuardHealthyStyle
		if !m.guardStatus.Active {
			healthStyle = GuardDegradedStyle
		}

		summary := []string{
			"Status: " + healthStyle.Render(m.guardStatus.Health),
			"Detail: " + m.guardStatus.Detail,
			"Mode: " + session.ModeDisplay(),
			"Context: " + session.TargetContext,
			"Namespace: " + session.NamespaceDisplay(),
			"Remaining: " + formatGuardDuration(m.guardStatus.Remaining),
			"Proxy: " + session.ProxyListenAddress,
			"Kubeconfig: " + session.GeneratedKubeconfigPath,
			"Expires: " + session.ExpiresAt.Format(time.RFC3339),
		}
		b.WriteString(GuardPanelStyle.Render(strings.Join(summary, "\n")))
	}

	if m.guardStatus != nil && len(m.guardStatus.RecentEvents) > 0 {
		recent := make([]string, 0, len(m.guardStatus.RecentEvents)+1)
		recent = append(recent, "Recent Events:")
		for _, event := range m.guardStatus.RecentEvents {
			recent = append(recent, fmt.Sprintf("%s | %s | %s",
				event.Timestamp.Format("15:04:05"),
				event.Type,
				event.Message,
			))
		}
		b.WriteString("\n\n")
		b.WriteString(GuardPanelStyle.Render(strings.Join(recent, "\n")))
	}

	b.WriteString("\n\n")
	b.WriteString(" " + DimItemStyle.Render("TTL preset: "+m.selectedGuardTTL().String()) + "\n\n")

	for index, action := range m.guardActions() {
		label := NormalItemStyle.Render(action)
		prefix := "  "
		if index == m.guardCursor {
			prefix = SelectedItemStyle.Render(IconCurrent) + " "
			label = SelectedItemStyle.Render(action)
		}
		b.WriteString(" " + prefix + label + "\n")
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

type contextAddedMsg struct {
	name string
	err  error
}

type contextRenamedMsg struct {
	oldName string
	newName string
	err     error
}

type contextRemovedMsg struct {
	name string
	err  error
}

type guardStatusLoadedMsg struct {
	status *application.GuardStatus
	err    error
}

type guardStartedMsg struct {
	session *domain.Session
	err     error
}

type guardStoppedMsg struct {
	session *domain.Session
	err     error
}

type guardTickMsg time.Time

type healthCheckCompleteMsg struct {
	results []healthdomain.Result
}

type healthCheckProgressMsg struct {
	result healthdomain.Result
}

type healthRefreshRequestMsg struct{}

type healthSpinnerTickMsg time.Time

type policiesLoadedMsg struct {
	policies        []domain.Policy
	policyUserFlags map[string]bool
	err             error
}

type groupsLoadedMsg struct {
	groups []domaingroup.Group
	err    error
}

type kubeconfigSourcesLoadedMsg struct {
	sources []application.KubeconfigSourceInfo
}

type kubeconfigSourceActivatedMsg struct {
	path string
	err  error
}

type groupUsedMsg struct {
	message string
	err     error
}

func (m Model) loadPolicies() tea.Msg {
	policies := m.policyService.ListPolicies()
	userProfiles := config.GetProfiles()
	flags := make(map[string]bool, len(policies))
	for _, p := range policies {
		if _, ok := userProfiles[p.Name]; ok {
			flags[p.Name] = true
		}
	}
	return policiesLoadedMsg{policies: policies, policyUserFlags: flags}
}

func (m Model) loadGroups() tea.Msg {
	groups, err := m.groupService.List()
	return groupsLoadedMsg{groups: groups, err: err}
}

func (m Model) loadKubeconfigSources() tea.Msg {
	sources := m.service.ListKubeconfigSources(config.GetKubeconfigPath(), config.GetKubeconfigSourceDirs())
	return kubeconfigSourcesLoadedMsg{sources: sources}
}

func (m Model) loadContexts() tea.Msg {
	kubeconfigPath := config.GetKubeconfigPath()
	contexts, err := m.service.ListContexts(kubeconfigPath)
	return contextsLoadedMsg{contexts: contexts, err: err}
}

func (m Model) loadGuardStatus() tea.Msg {
	status, err := m.guardService.Status()
	return guardStatusLoadedMsg{status: status, err: err}
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

func (m Model) activateKubeconfigSource(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.service.ValidateKubeconfigSource(path); err != nil {
			return kubeconfigSourceActivatedMsg{path: path, err: err}
		}
		if err := config.SetActiveKubeconfigPath(path); err != nil {
			return kubeconfigSourceActivatedMsg{path: path, err: err}
		}
		return kubeconfigSourceActivatedMsg{path: config.GetKubeconfigPath()}
	}
}

func (m *Model) rebuildKubeconfigScopedServices(kubeconfigPath string) {
	config.SetKubeconfigPath(kubeconfigPath)
	repo := infrastructure.NewFileRepository()
	m.groupService = appgroupservice.NewService(
		groupstore.NewFileStore(config.GetGroupsPath()),
		repo,
		kubeconfigPath,
		appgroupservice.WithPolicyResolver(m.policyService),
	)
	m.healthService = healthservice.New(
		healthcheck.New(kubeconfigPath),
		healthcheck.NewCache(),
		repo,
		kubeconfigPath,
	)
	m.healthResults = make(map[string]healthdomain.Result)
	m.healthChecking = false
	m.healthPending = 0
	m.healthLoaded = false
}

func (m Model) useGroup(g domaingroup.Group) tea.Cmd {
	return func() tea.Msg {
		resolved, missing, err := m.groupService.Resolve(g.Name)
		if err != nil {
			return groupUsedMsg{err: err}
		}

		contexts, err := m.service.ListContexts(config.GetKubeconfigPath())
		if err != nil {
			return groupUsedMsg{err: err}
		}

		available := groupContextInfos(resolved, contexts, missing)
		if len(available) == 0 {
			return groupUsedMsg{err: fmt.Errorf("group %q has no contexts present in your kubeconfig", resolved.Name)}
		}

		selected := available[0].Name
		var session *domain.Session
		var previousPolicy string
		var hadActiveGuard bool
		if resolved.Policy != "" {
			status, err := m.guardService.Status()
			if err != nil {
				return groupUsedMsg{err: err}
			}
			previousPolicy, hadActiveGuard = activeGuardPolicyName(status)
			session, err = m.guardService.StartReadonly(application.GuardStartOptions{
				SourcePath:    config.GetKubeconfigPath(),
				Profile:       resolved.Policy,
				TargetContext: selected,
				ReplaceActive: true,
			})
			if err != nil {
				return groupUsedMsg{err: fmt.Errorf("activate guard for group %q with policy %q: %w", resolved.Name, resolved.Policy, err)}
			}
		}

		if err := m.service.UseContext(config.GetKubeconfigPath(), selected, ""); err != nil {
			if session != nil {
				_, _ = m.guardService.Stop()
			}
			return groupUsedMsg{err: err}
		}

		message := fmt.Sprintf("Switched to '%s' from group '%s'", selected, resolved.Name)
		if session != nil {
			message = fmt.Sprintf("%s; Guard active: %s", message, session.PolicyName)
			if hadActiveGuard && previousPolicy != session.PolicyName {
				message = fmt.Sprintf("Guard policy changed from %s to %s because group %s requires it; %s", previousPolicy, session.PolicyName, resolved.Name, message)
			}
		}
		return groupUsedMsg{message: message}
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

func (m Model) addContext(sourcePath, contextName string) tea.Cmd {
	return func() tea.Msg {
		kubeconfigPath := config.GetKubeconfigPath()
		if err := m.service.AddConfig(sourcePath, kubeconfigPath, contextName); err != nil {
			return contextAddedMsg{name: contextName, err: err}
		}
		return contextAddedMsg{name: contextName, err: nil}
	}
}

func (m Model) renameContext(oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		kubeconfigPath := config.GetKubeconfigPath()
		if err := m.service.RenameContext(kubeconfigPath, oldName, newName); err != nil {
			return contextRenamedMsg{oldName: oldName, newName: newName, err: err}
		}
		return contextRenamedMsg{oldName: oldName, newName: newName, err: nil}
	}
}

func (m Model) removeContext(name string) tea.Cmd {
	return func() tea.Msg {
		kubeconfigPath := config.GetKubeconfigPath()
		if err := m.service.RemoveContext(kubeconfigPath, name); err != nil {
			return contextRemovedMsg{name: name, err: err}
		}
		return contextRemovedMsg{name: name, err: nil}
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

func (m Model) startGuard(ttl time.Duration) tea.Cmd {
	return func() tea.Msg {
		session, err := m.guardService.StartReadonly(application.GuardStartOptions{
			SourcePath: config.GetKubeconfigPath(),
			TTL:        ttl,
		})
		return guardStartedMsg{session: session, err: err}
	}
}

func (m Model) stopGuard() tea.Cmd {
	return func() tea.Msg {
		session, err := m.guardService.Stop()
		return guardStoppedMsg{session: session, err: err}
	}
}

func (m Model) selectGuardAction() tea.Cmd {
	actions := m.guardActions()
	if len(actions) == 0 {
		return nil
	}

	switch actions[m.guardCursor] {
	case "Start Readonly Guard":
		return m.startGuard(m.selectedGuardTTL())
	case "Stop Guard":
		return m.stopGuard()
	default:
		return m.loadGuardStatus
	}
}

func (m Model) guardActions() []string {
	if m.guardStatus != nil && m.guardStatus.Active {
		return []string{"Stop Guard", "Refresh Status"}
	}
	return []string{"Start Readonly Guard", "Refresh Status"}
}

func (m Model) selectedGuardTTL() time.Duration {
	if len(m.guardTTLOptions) == 0 {
		return 30 * time.Minute
	}
	return m.guardTTLOptions[m.guardTTLIndex]
}

func tickGuardStatus() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return guardTickMsg(t)
	})
}

var healthSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type confirmPollMsg time.Time

type confirmPendingMsg struct {
	pending *domain.PendingConfirmation
}

func tickConfirmPoll() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return confirmPollMsg(t)
	})
}

func tickHealthSpinner() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return healthSpinnerTickMsg(t)
	})
}

func (m *Model) startHealthChecksFor(names []string, force bool) tea.Cmd {
	if m.healthService == nil || len(names) == 0 {
		return nil
	}

	cmds := make([]tea.Cmd, 0, len(names)+1)
	seen := make(map[string]struct{}, len(names))

	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		if !force {
			if result := m.healthService.GetCached(name); result.Status != healthdomain.StatusUnknown {
				m.healthResults[name] = result
				continue
			}
		}

		contextName := name
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := m.healthService.CheckContext(ctx, contextName)
			return healthCheckProgressMsg{result: result}
		})
	}

	if len(cmds) == 0 {
		m.healthLoaded = true
		return nil
	}

	m.healthChecking = true
	m.healthPending = len(cmds)
	cmds = append(cmds, tickHealthSpinner())
	return tea.Batch(cmds...)
}

func (m Model) healthIndicator(contextName string) (string, string) {
	result, ok := m.healthResults[contextName]
	if !ok {
		if m.healthChecking {
			return HealthUnknownStyle.Render("?"), DimItemStyle.Render("(checking... " + healthSpinnerFrames[m.healthSpinner] + ")")
		}
		return HealthUnknownStyle.Render("?"), ""
	}

	switch result.Status {
	case healthdomain.StatusHealthy:
		return HealthHealthyStyle.Render(result.Status.Emoji()), DimItemStyle.Render("(" + formatHealthLatency(result.Latency) + ")")
	case healthdomain.StatusDegraded:
		return HealthDegradedStyle.Render(result.Status.Emoji()), DimItemStyle.Render("(" + formatHealthLatency(result.Latency) + ")")
	case healthdomain.StatusUnhealthy, healthdomain.StatusUnreachable:
		return HealthUnhealthyStyle.Render(result.Status.Emoji()), ""
	default:
		if m.healthChecking {
			return HealthUnknownStyle.Render("?"), DimItemStyle.Render("(checking... " + healthSpinnerFrames[m.healthSpinner] + ")")
		}
		return HealthUnknownStyle.Render("?"), ""
	}
}

func formatHealthLatency(latency time.Duration) string {
	if latency <= 0 {
		return "—"
	}
	if latency < time.Second {
		return latency.Round(time.Millisecond).String()
	}
	return latency.Round(100 * time.Millisecond).String()
}

func contextNames(contexts []application.ContextInfo) []string {
	names := make([]string, 0, len(contexts))
	for _, contextInfo := range contexts {
		names = append(names, contextInfo.Name)
	}
	return names
}

func groupContextInfos(g domaingroup.Group, contexts []application.ContextInfo, missing []string) []application.ContextInfo {
	missingSet := make(map[string]struct{}, len(missing))
	for _, name := range missing {
		missingSet[name] = struct{}{}
	}

	byName := make(map[string]application.ContextInfo, len(contexts))
	for _, context := range contexts {
		byName[context.Name] = context
	}

	filtered := make([]application.ContextInfo, 0, len(g.Contexts))
	for _, contextName := range g.Contexts {
		if _, isMissing := missingSet[contextName]; isMissing {
			continue
		}
		context, ok := byName[contextName]
		if ok {
			filtered = append(filtered, context)
		}
	}

	return filtered
}

func activeGuardPolicyName(status *application.GuardStatus) (string, bool) {
	if status == nil || !status.Active || status.Session == nil {
		return "", false
	}
	if status.Session.PolicyName != "" {
		return status.Session.PolicyName, true
	}
	return "readonly", true
}

func formatGuardDuration(value time.Duration) string {
	if value <= 0 {
		return "expired"
	}
	return value.Round(time.Second).String()
}

func Run() error {
	config.Init()
	model, err := NewModel(nil)
	if err != nil {
		return err
	}
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

func RunWithConfig(kubeconfigPath string, healthSvc *healthservice.Service) error {
	config.Init()
	config.SetKubeconfigPath(kubeconfigPath)
	model, err := NewModel(healthSvc)
	if err != nil {
		return err
	}
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}
