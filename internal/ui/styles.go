package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	primaryColor  = lipgloss.Color("#00D9FF")
	successColor  = lipgloss.Color("#00FF87")
	errorColor    = lipgloss.Color("#FF5F87")
	warningColor  = lipgloss.Color("#FFD700")
	infoColor     = lipgloss.Color("#87CEEB")
	subtleColor   = lipgloss.Color("#666666")
	activeColor   = lipgloss.Color("#FFD700")
	inactiveColor = lipgloss.Color("#6C7086")
	currentColor  = lipgloss.Color("#89B4FA")
	headerColor   = lipgloss.Color("#CBA6F7")
	borderColor   = lipgloss.Color("#45475A")
)

const (
	IconSuccess   = "✓"
	IconError     = "✗"
	IconWarning   = "⚠"
	IconInfo      = "ℹ"
	IconCurrent   = "→"
	IconContext   = "⎈"
	IconNamespace = "◉"
	IconCluster   = "⚙"
	IconServer    = "🌐"
	IconCheck     = "✓"
	IconCross     = "✗"
)

var (
	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			PaddingLeft(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			PaddingLeft(1)

	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true).
			PaddingLeft(1)

	InfoStyle = lipgloss.NewStyle().
			Foreground(infoColor).
			PaddingLeft(1)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(true).
			Underline(true)

	CurrentIndicatorStyle = lipgloss.NewStyle().
				Foreground(currentColor).
				Bold(true)

	ContextNameStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	ClusterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1"))

	ServerStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	NamespaceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9E2AF"))

	BorderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	ActiveItemStyle = lipgloss.NewStyle().
			Foreground(activeColor).
			Bold(true).
			PaddingLeft(1)

	InactiveItemStyle = lipgloss.NewStyle().
				Foreground(inactiveColor).
				PaddingLeft(1)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	LabelStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(primaryColor)
)

func Success(message string) string {
	return SuccessStyle.Render(IconSuccess + " " + message)
}

func Error(message string) string {
	return ErrorStyle.Render(IconError + " " + message)
}

func Warning(message string) string {
	return WarningStyle.Render(IconWarning + " " + message)
}

func Info(message string) string {
	return InfoStyle.Render(IconInfo + " " + message)
}

func Header(text string) string {
	return HeaderStyle.Render(text)
}

func ContextName(name string) string {
	return ContextNameStyle.Render(name)
}

func CurrentIndicator() string {
	return CurrentIndicatorStyle.Render(IconCurrent)
}

func Cluster(name string) string {
	return ClusterStyle.Render(name)
}

func Server(url string) string {
	return ServerStyle.Render(url)
}

func Namespace(name string) string {
	return NamespaceStyle.Render(name)
}

func Label(text string) string {
	return LabelStyle.Render(text)
}

func Value(text string) string {
	return ValueStyle.Render(text)
}

func Border(content string) string {
	return BorderStyle.Render(content)
}
