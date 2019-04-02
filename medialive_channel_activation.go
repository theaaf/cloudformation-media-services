package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/medialive"
)

func init() {
	RegisterType("Custom::MediaLiveChannelActivation", MediaLiveChannelActivation)
}

func MediaLiveChannelActivation(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	client := medialive.New(cfg)

	switch request.RequestType {
	case "Create", "Update":
		var input medialive.StartChannelInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		if _, err := client.StartChannelRequest(&input).Send(); err != nil {
			return nil, err
		}
		if err := waitForChannelState(client, *input.ChannelId, medialive.ChannelStateRunning, []medialive.ChannelState{
			medialive.ChannelStateStarting,
		}); err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: *input.ChannelId + "/activation",
		}, nil
	case "Delete":
		var input medialive.StopChannelInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		if _, err := client.StopChannelRequest(&input).Send(); err != nil {
			return nil, err
		}
		if err := waitForChannelState(client, *input.ChannelId, medialive.ChannelStateIdle, []medialive.ChannelState{
			medialive.ChannelStateStopping,
		}); err != nil {
			return nil, err
		}
		return nil, nil
	}

	return nil, fmt.Errorf("unexpected request type")
}
