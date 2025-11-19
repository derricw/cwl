package model

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Config centralizes all configuration
type Config struct {
	Styles   StyleConfig
	KeyBinds KeyBindConfig
	Timeouts TimeoutConfig
}

// StyleConfig holds all styling configuration
type StyleConfig struct {
	DocStyle   lipgloss.Style
	ErrorStyle lipgloss.Style
}

// KeyBindConfig holds key binding configuration
type KeyBindConfig struct {
	Quit   string
	Back   string
	Select string
}

// TimeoutConfig holds timeout configuration
type TimeoutConfig struct {
	APITimeout time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Styles: StyleConfig{
			DocStyle:   lipgloss.NewStyle().Margin(1, 2),
			ErrorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		},
		KeyBinds: KeyBindConfig{
			Quit:   "ctrl+c",
			Back:   "esc",
			Select: "enter",
		},
		Timeouts: TimeoutConfig{
			APITimeout: 30 * time.Second,
		},
	}
}