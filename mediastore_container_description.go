package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mediastore"
)

func init() {
	RegisterType("Custom::MediaStoreContainerDescription", MediaStoreContainerDescription)
}

func MediaStoreContainerDescription(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	client := mediastore.New(cfg)

	switch request.RequestType {
	case "Create", "Update":
		var input mediastore.DescribeContainerInput
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}
		resp, err := client.DescribeContainerRequest(&input).Send()
		if err != nil {
			return nil, err
		}
		return &Success{
			PhysicalResourceId: "MediaStoreContainerDescription/" + *resp.Container.ARN,
			Data: map[string]interface{}{
				"Arn":      *resp.Container.ARN,
				"Endpoint": *resp.Container.Endpoint,
			},
		}, nil
	case "Delete":
		return nil, nil
	}

	return nil, fmt.Errorf("unexpected request type")
}
