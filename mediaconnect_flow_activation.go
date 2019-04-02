package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mediaconnect"
)

func init() {
	RegisterType("Custom::MediaConnectFlowActivation", MediaConnectFlowActivation)
}

func waitForFlowStatus(client *mediaconnect.MediaConnect, flowArn string, status mediaconnect.Status, allowed []mediaconnect.Status) error {
	for {
		resp, err := client.DescribeFlowRequest(&mediaconnect.DescribeFlowInput{
			FlowArn: &flowArn,
		}).Send()
		if err != nil {
			return err
		}
		if resp.Flow.Status == status {
			return nil
		}
		isUnexpectedState := true
		for _, status := range allowed {
			if resp.Flow.Status == status {
				isUnexpectedState = false
				break
			}
		}
		if isUnexpectedState {
			return fmt.Errorf("Channel reached unexpected state: " + string(resp.Flow.Status))
		}
		time.Sleep(5 * time.Second)
	}
}

func MediaConnectFlowActivation(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	client := mediaconnect.New(cfg)

	switch request.RequestType {
	case "Create", "Update":
		var input mediaconnect.StartFlowInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		if _, err := client.StartFlowRequest(&input).Send(); err != nil {
			return nil, err
		}
		if err := waitForFlowStatus(client, *input.FlowArn, mediaconnect.StatusActive, []mediaconnect.Status{
			mediaconnect.StatusStarting,
		}); err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: *input.FlowArn + "/activation",
		}, nil
	case "Delete":
		var input mediaconnect.StopFlowInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		if _, err := client.StopFlowRequest(&input).Send(); err != nil {
			return nil, err
		}
		if err := waitForFlowStatus(client, *input.FlowArn, mediaconnect.StatusStandby, []mediaconnect.Status{
			mediaconnect.StatusStopping,
		}); err != nil {
			return nil, err
		}
		return nil, nil
	}

	return nil, fmt.Errorf("unexpected request type")
}
