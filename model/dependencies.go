package model

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/derricw/cwl/fetch"
)

// Dependencies holds all external dependencies
type Dependencies struct {
	Profile string
	Client  *cloudwatchlogs.Client
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