package main

import (
	"crypto/rand"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func init() {
	RegisterType("Custom::RandomString", RandomString)
}

func RandomString(request *CustomResourceRequest, cfg aws.Config) (*Success, error) {
	switch request.RequestType {
	case "Create", "Update":
		var input struct{}
		if err := ReshapeProps(request.ResourceProperties, &input); err != nil {
			return nil, err
		}

		// Crockford's base32 encoding: https://www.crockford.com/wrmg/base32.html
		b := make([]byte, 12)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		ret := ""
		const chars = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
		for _, b := range b {
			ret += string(chars[int(b)%len(chars)])
		}

		return &Success{
			PhysicalResourceId: ret,
			Data: map[string]interface{}{
				"String": ret,
			},
		}, nil
	case "Delete":
		return nil, nil
	}

	return nil, fmt.Errorf("unexpected request type")
}
