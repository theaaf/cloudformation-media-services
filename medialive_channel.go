package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/medialive"
)

func init() {
	RegisterType("Custom::MediaLiveChannel", MediaLiveChannel)
}

func waitForChannelState(client *medialive.MediaLive, channelId string, state medialive.ChannelState, allowed []medialive.ChannelState) error {
	for {
		resp, err := client.DescribeChannelRequest(&medialive.DescribeChannelInput{
			ChannelId: &channelId,
		}).Send()
		if err != nil {
			return err
		}
		if resp.State == state {
			return nil
		}
		isUnexpectedState := true
		for _, state := range allowed {
			if resp.State == state {
				isUnexpectedState = false
				break
			}
		}
		if isUnexpectedState {
			return fmt.Errorf("Channel reached unexpected state: " + string(resp.State))
		}
		time.Sleep(5 * time.Second)
	}
}

func MediaLiveChannel(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	client := medialive.New(cfg)

	switch request.RequestType {
	case "Create":
		var input medialive.CreateChannelInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		resp, err := client.CreateChannelRequest(&input).Send()
		if err != nil {
			return nil, err
		}
		if err := waitForChannelState(client, *resp.Channel.Id, medialive.ChannelStateIdle, []medialive.ChannelState{
			medialive.ChannelStateCreating,
		}); err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: *resp.Channel.Id,
			Data: map[string]interface{}{
				"Arn": *resp.Channel.Arn,
				"Id":  *resp.Channel.Id,
			},
		}, nil
	case "Update":
		var input medialive.UpdateChannelInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		input.ChannelId = &request.PhysicalResourceId
		resp, err := client.UpdateChannelRequest(&input).Send()
		if err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: *resp.Channel.Id,
			Data: map[string]interface{}{
				"Arn": *resp.Channel.Arn,
				"Id":  *resp.Channel.Id,
			},
		}, nil
	case "Delete":
		if _, err := client.DeleteChannelRequest(&medialive.DeleteChannelInput{
			ChannelId: &request.PhysicalResourceId,
		}).Send(); err != nil {
			return nil, err
		}
		if err := waitForChannelState(client, request.PhysicalResourceId, medialive.ChannelStateDeleted, []medialive.ChannelState{
			medialive.ChannelStateDeleting,
		}); err != nil {
			return nil, err
		}
		return nil, nil
	}

	return nil, fmt.Errorf("unexpected request type")
}
