package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mediaconnect"
)

func init() {
	RegisterType("Custom::MediaConnectFlow", MediaConnectFlow)
}

func createMediaConnectFlow(props map[string]interface{}, client *mediaconnect.MediaConnect) (*Success, error) {
	var input mediaconnect.CreateFlowInput
	if err := ReshapeProps(props, &input); err != nil {
		return nil, err
	}
	resp, err := client.CreateFlowRequest(&input).Send()
	if err != nil {
		return nil, err
	}
	return &Success{
		PhysicalResourceId: *resp.Flow.FlowArn,
		Data: map[string]interface{}{
			"Arn": *resp.Flow.FlowArn,
		},
	}, nil
}

func MediaConnectFlow(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	client := mediaconnect.New(cfg)

	switch request.RequestType {
	case "Create", "Update":
		return createMediaConnectFlow(request.ResourceProperties, client)
	case "Delete":
		_, err := client.DeleteFlowRequest(&mediaconnect.DeleteFlowInput{
			FlowArn: &request.PhysicalResourceId,
		}).Send()
		return nil, err
	}

	return nil, fmt.Errorf("unexpected request type")
}
