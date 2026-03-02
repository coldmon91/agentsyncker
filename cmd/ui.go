package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	styleSuccess = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	styleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	styleError   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func printTitle(w io.Writer, title string) {
	_, _ = fmt.Fprintln(w, styleTitle.Render(title))
	_, _ = fmt.Fprintln(w, styleDim.Render(strings.Repeat("─", lipgloss.Width(title))))
}

func printSuccess(w io.Writer, message string) {
	_, _ = fmt.Fprintln(w, styleSuccess.Render("✔ "+message))
}

func printInfo(w io.Writer, message string) {
	_, _ = fmt.Fprintln(w, styleInfo.Render("ℹ "+message))
}

func printError(w io.Writer, message string) {
	_, _ = fmt.Fprintln(w, styleError.Render("✘ "+message))
}

func printResultRow(w io.Writer, label string, value string) {
	_, _ = fmt.Fprintf(w, "%s %s\n", styleDim.Render(label+":"), value)
}
