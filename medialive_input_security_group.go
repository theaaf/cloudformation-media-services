package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/medialive"
)

func init() {
	RegisterType("Custom::MediaLiveInputSecurityGroup", MediaLiveInputSecurityGroup)
}

func MediaLiveInputSecurityGroup(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	client := medialive.New(cfg)

	switch request.RequestType {
	case "Create":
		var input medialive.CreateInputSecurityGroupInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		resp, err := client.CreateInputSecurityGroupRequest(&input).Send()
		if err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: *resp.SecurityGroup.Id,
			Data: map[string]interface{}{
				"Arn": *resp.SecurityGroup.Arn,
				"Id":  *resp.SecurityGroup.Id,
			},
		}, nil
	case "Update":
		var input medialive.UpdateInputSecurityGroupInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		input.InputSecurityGroupId = &request.PhysicalResourceId
		resp, err := client.UpdateInputSecurityGroupRequest(&input).Send()
		if err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: *resp.SecurityGroup.Id,
			Data: map[string]interface{}{
				"Arn": *resp.SecurityGroup.Arn,
				"Id":  *resp.SecurityGroup.Id,
			},
		}, nil
	case "Delete":
		_, err := client.DeleteInputSecurityGroupRequest(&medialive.DeleteInputSecurityGroupInput{
			InputSecurityGroupId: &request.PhysicalResourceId,
		}).Send()
		return nil, err
	}

	return nil, fmt.Errorf("unexpected request type")
}
