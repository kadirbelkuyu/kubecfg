package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("#89B4FA")
	secondaryColor = lipgloss.Color("#CBA6F7")
	accentColor    = lipgloss.Color("#F9E2AF")
	successColor   = lipgloss.Color("#A6E3A1")
	errorColor     = lipgloss.Color("#F38BA8")
	textColor      = lipgloss.Color("#CDD6F4")
	subtleColor    = lipgloss.Color("#6C7086")
	dimColor       = lipgloss.Color("#45475A")
	cyanColor      = lipgloss.Color("#4ECDC4")
	cyanDarkColor  = lipgloss.Color("#2EC4B6")
)

var (
	AppNameStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	LogoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9C0")).
			Bold(true)

	DescStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(textColor)

	DimItemStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	CurrentMarkerStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	ContextNameStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	ClusterStyle = lipgloss.NewStyle().
			Foreground(successColor)

	NamespaceStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	ServerStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimColor).
			Padding(1, 2)

	TitleBarStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			MarginTop(1)

	FilterInputStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Background(lipgloss.Color("#313244")).
				Padding(0, 1)
)

const (
	IconContext   = "⎈"
	IconNamespace = "⬡"
	IconCluster   = "◆"
	IconServer    = "●"
	IconCurrent   = "❯"
	IconCheck     = "✓"
	IconCross     = "✗"
	IconMenu      = "☰"
	IconSwitch    = "⇄"
	IconList      = "≡"
	IconInfo      = "ℹ"
	IconExit      = "⏻"
	IconAdd       = "+"
	IconRename    = "✎"
	IconRemove    = "−"
)

type MenuItem struct {
	Icon  string
	Label string
}

var MenuItems = []MenuItem{
	{Icon: IconSwitch, Label: "Switch Context"},
	{Icon: IconNamespace, Label: "Set Namespace"},
	{Icon: IconAdd, Label: "Add Context"},
	{Icon: IconRename, Label: "Rename Context"},
	{Icon: IconRemove, Label: "Remove Context"},
	{Icon: IconInfo, Label: "Current Info"},
	{Icon: IconExit, Label: "Exit"},
}
