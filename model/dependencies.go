package model

import (
	"github.com/derricw/cwl/fetch"
	"github.com/derricw/cwl/interfaces"
)

// Dependencies holds all external dependencies
type Dependencies struct {
	Profile string
	Client  interfaces.CloudWatchLogsClient
}

// NewDependencies creates dependencies with AWS client
func NewDependencies(profile string) (*Dependencies, error) {
	client, err := fetch.CreateClient(profile)
	if err != nil {
		return nil, err
	}
	
	return &Dependencies{
		Profile: profile,
		Client:  client,
	}, nil
}