package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("#00D9FF")
	secondaryColor = lipgloss.Color("#CBA6F7")
	successColor   = lipgloss.Color("#00FF87")
	errorColor     = lipgloss.Color("#FF5F87")
	warningColor   = lipgloss.Color("#FFD700")
	subtleColor    = lipgloss.Color("#6C7086")
	highlightColor = lipgloss.Color("#F9E2AF")
	borderColor    = lipgloss.Color("#45475A")
	bgColor        = lipgloss.Color("#1E1E2E")
)

var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true)

	MenuTitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Padding(0, 1)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(highlightColor).
				Bold(true).
				PaddingLeft(2)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			PaddingLeft(4)

	CurrentMarkerStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginTop(1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Background(lipgloss.Color("#313244")).
			Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Underline(true).
			MarginBottom(1)

	ContextNameStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	ClusterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1"))

	NamespaceStyle = lipgloss.NewStyle().
			Foreground(highlightColor)

	ServerStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)
)

const (
	IconContext   = "⎈"
	IconNamespace = "◉"
	IconCluster   = "⚙"
	IconServer    = "🌐"
	IconCurrent   = "→"
	IconCheck     = "✓"
	IconCross     = "✗"
	IconMenu      = "☰"
)
