package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/medialive"
)

func init() {
	RegisterType("Custom::MediaLiveInput", MediaLiveInput)
}

func createMediaLiveInput(props map[string]interface{}, client *medialive.MediaLive) (*Success, error) {
	var input medialive.CreateInputInput
	if err := ReshapeProps(props, &input); err != nil {
		return nil, err
	}
	resp, err := client.CreateInputRequest(&input).Send()
	if err != nil {
		return nil, err
	}
	return &Success{
		PhysicalResourceId: *resp.Input.Id,
		Data: map[string]interface{}{
			"Arn": *resp.Input.Arn,
			"Id":  *resp.Input.Id,
		},
	}, nil
}

func MediaLiveInput(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	client := medialive.New(cfg)

	switch request.RequestType {
	case "Create":
		return createMediaLiveInput(request.ResourceProperties, client)
	case "Update":
		needsReplacement := false

		updateProps := make(map[string]interface{})
		for k, v := range request.ResourceProperties {
			switch k {
			case "Type":
				if v != request.OldResourceProperties[k] {
					needsReplacement = true
				}
			default:
				updateProps[k] = v
			}
		}

		if needsReplacement {
			return createMediaLiveInput(request.ResourceProperties, client)
		}

		var input medialive.UpdateInputInput
		if err := ReshapeProps(updateProps, &input); err != nil {
			return nil, err
		}
		input.InputId = &request.PhysicalResourceId
		resp, err := client.UpdateInputRequest(&input).Send()
		if err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: *resp.Input.Id,
			Data: map[string]interface{}{
				"Arn": *resp.Input.Arn,
				"Id":  *resp.Input.Id,
			},
		}, nil
	case "Delete":
		_, err := client.DeleteInputRequest(&medialive.DeleteInputInput{
			InputId: &request.PhysicalResourceId,
		}).Send()
		return nil, err
	}

	return nil, fmt.Errorf("unexpected request type")
}
