package model

import (
	"github.com/derricw/cwl/provider"
)

// Dependencies holds all external dependencies for the TUI.
// Backend is the log provider (CloudWatch, MLflow, etc.).
type Dependencies struct {
	Profile string
	Backend provider.Backend
}
