package cmd

import "github.com/charmbracelet/lipgloss"

var colors []lipgloss.Color = []lipgloss.Color{
	// Whites and Grays
	lipgloss.Color("#FFFFFF"), // Pure White
	lipgloss.Color("#EEEEEE"), // Bright White
	lipgloss.Color("#DDDDDD"), // Light Gray
	lipgloss.Color("#CCCCCC"), // Pale Gray

	// Reds & Pinks
	lipgloss.Color("#FF0000"), // Pure Red
	lipgloss.Color("#FF3333"), // Bright Red
	lipgloss.Color("#FF6666"), // Light Red
	lipgloss.Color("#FF0066"), // Hot Pink
	lipgloss.Color("#FF66B2"), // Rose Pink
	lipgloss.Color("#FF99CC"), // Pastel Pink
	lipgloss.Color("#FF0099"), // Magenta Pink
	lipgloss.Color("#FF33FF"), // Bright Magenta

	// Oranges
	lipgloss.Color("#FF8000"), // Pure Orange
	lipgloss.Color("#FFB266"), // Light Orange
	lipgloss.Color("#FF9933"), // Dark Orange
	lipgloss.Color("#FFAA00"), // Amber
	lipgloss.Color("#FFB84D"), // Golden Orange
	lipgloss.Color("#FFCC00"), // Golden Yellow

	// Yellows
	lipgloss.Color("#FFFF00"), // Pure Yellow
	lipgloss.Color("#FFFF66"), // Light Yellow
	lipgloss.Color("#FFFF99"), // Pale Yellow
	lipgloss.Color("#FFED4D"), // Warm Yellow
	lipgloss.Color("#FFE135"), // Banana Yellow

	// Greens
	lipgloss.Color("#00FF00"), // Pure Green
	lipgloss.Color("#33FF33"), // Bright Green
	lipgloss.Color("#66FF66"), // Light Green
	lipgloss.Color("#00FF66"), // Spring Green
	lipgloss.Color("#00FF99"), // Mint Green
	lipgloss.Color("#33FF99"), // Sea Green
	lipgloss.Color("#66FFB2"), // Pale Green
	lipgloss.Color("#99FF99"), // Soft Green
	lipgloss.Color("#CCFF00"), // Lime Green
	lipgloss.Color("#B2FF66"), // Yellow Green

	// Blues
	lipgloss.Color("#00FFFF"), // Pure Cyan
	lipgloss.Color("#33FFFF"), // Bright Cyan
	lipgloss.Color("#66FFFF"), // Light Cyan
	lipgloss.Color("#00CCFF"), // Sky Blue
	lipgloss.Color("#0099FF"), // Azure Blue
	lipgloss.Color("#66B2FF"), // Light Blue
	lipgloss.Color("#99CCFF"), // Pale Blue
	lipgloss.Color("#66FFFF"), // Electric Blue

	// Purples
	lipgloss.Color("#FF00FF"), // Pure Magenta
	lipgloss.Color("#FF33CC"), // Bright Pink
	lipgloss.Color("#CC66FF"), // Light Purple
	lipgloss.Color("#9933FF"), // Purple
	lipgloss.Color("#B266FF"), // Lavender
	lipgloss.Color("#CC99FF"), // Pale Purple

	// Special and Mixed Colors
	lipgloss.Color("#FFB2B2"), // Salmon
	lipgloss.Color("#FFD700"), // Gold
	lipgloss.Color("#B2FFB2"), // Mint Cream
	lipgloss.Color("#B2B2FF"), // Periwinkle
	lipgloss.Color("#E6B3FF"), // Light Orchid
	lipgloss.Color("#FFE6B3"), // Peach
	lipgloss.Color("#B3FFE6"), // Aquamarine
	lipgloss.Color("#E6FFB3"), // Light Chartreuse
}
