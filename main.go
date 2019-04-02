package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

type CustomResourceRequest struct {
	RequestType           string                 `json:"RequestType"`
	ResponseURL           string                 `json:"ResponseURL"`
	StackId               string                 `json:"StackId"`
	RequestId             string                 `json:"RequestId"`
	ResourceType          string                 `json:"ResourceType"`
	LogicalResourceId     string                 `json:"LogicalResourceId"`
	PhysicalResourceId    string                 `json:"PhysicalResourceId"`
	ResourceProperties    map[string]interface{} `json:"ResourceProperties"`
	OldResourceProperties map[string]interface{} `json:"OldResourceProperties"`
}

type CustomResourceResponse struct {
	Status             string                 `json:"Status"`
	Reason             string                 `json:"Reason"`
	PhysicalResourceId string                 `json:"PhysicalResourceId"`
	StackId            string                 `json:"StackId"`
	RequestId          string                 `json:"RequestId"`
	LogicalResourceId  string                 `json:"LogicalResourceId"`
	NoEcho             bool                   `json:"NoEcho,omitempty"`
	Data               map[string]interface{} `json:"Data,omitempty"`
}

type Success struct {
	PhysicalResourceId string                 `json:"PhysicalResourceId"`
	Data               map[string]interface{} `json:"Data,omitempty"`
}

func ReshapeProps(in map[string]interface{}, out interface{}) error {
	if len(in) > 0 {
		filtered := make(map[string]interface{})
		for k, v := range in {
			if k != "ServiceToken" {
				filtered[k] = v
			}
		}
		in = filtered
	}
	return reshape(in, reflect.ValueOf(out), "")
}

func reshape(in interface{}, out reflect.Value, path string) error {
	if out.Kind() == reflect.Ptr {
		if out.IsNil() {
			out.Set(reflect.New(out.Type().Elem()))
		}
		return reshape(in, out.Elem(), path)
	}

	switch in := in.(type) {
	case bool:
		out.SetBool(in)
	case int:
		switch out.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			out.SetInt(int64(in))
		}
	case string:
		switch out.Kind() {
		case reflect.Bool:
			switch in {
			case "true":
				out.SetBool(true)
			case "false":
				out.SetBool(false)
			default:
				return fmt.Errorf("Invalid boolean value: %s", in)
			}
		case reflect.String:
			out.SetString(in)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if n, err := strconv.ParseInt(in, 10, 64); err == nil {
				out.SetInt(n)
			}
		}
	case []interface{}:
		if out.Kind() == reflect.Slice {
			slice := reflect.MakeSlice(out.Type(), len(in), len(in))
			for i, v := range in {
				if err := reshape(v, slice.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
			out.Set(slice)
		}
	case map[string]interface{}:
		for k, v := range in {
			if field := out.FieldByName(k); field.IsValid() {
				if err := reshape(v, field, path+"."+k); err != nil {
					return err
				}
			} else if path == "" {
				return fmt.Errorf("Unsupported property: %s", k)
			} else {
				return fmt.Errorf("Unsupported property: %s in %s", k, path)
			}
		}
	}

	return nil
}

var customResourceTypes = map[string]func(*CustomResourceRequest, aws.Config) (*Success, error){}

func RegisterType(name string, f func(*CustomResourceRequest, aws.Config) (*Success, error)) {
	customResourceTypes[name] = f
}

func Handler(request *CustomResourceRequest) {
	logrus.WithFields(logrus.Fields{
		"request_type":         request.RequestType,
		"response_url":         request.ResponseURL,
		"stack_id":             request.StackId,
		"request_id":           request.RequestId,
		"logical_resource_id":  request.LogicalResourceId,
		"physical_resource_id": request.PhysicalResourceId,
	}).Info("request received")

	var success *Success
	var err error

	defer func() {
		resp := &CustomResourceResponse{
			StackId:            request.StackId,
			RequestId:          request.RequestId,
			LogicalResourceId:  request.LogicalResourceId,
			PhysicalResourceId: request.PhysicalResourceId,
		}

		if r := recover(); r != nil {
			logrus.Error(r)
			logrus.Info(string(debug.Stack()))
			resp.Status = "FAILED"
			resp.Reason = "Handler panicked. See logs for details."
		} else if err != nil {
			resp.Status = "FAILED"
			resp.Reason = err.Error()
		} else {
			resp.Status = "SUCCESS"
			if success != nil {
				resp.PhysicalResourceId = success.PhysicalResourceId
				resp.Data = success.Data
			}
		}

		if request.RequestType == "Create" && resp.Status == "FAILED" && resp.PhysicalResourceId == "" {
			buf := make([]byte, 16)
			rand.Read(buf)
			resp.PhysicalResourceId = "failed/" + hex.EncodeToString(buf)
		}

		logrus.WithFields(logrus.Fields{
			"status": resp.Status,
			"reason": resp.Reason,
		}).Info("sending response")

		if b, err := json.Marshal(resp); err != nil {
			logrus.Error(err)
		} else if req, err := http.NewRequest("PUT", request.ResponseURL, bytes.NewReader(b)); err != nil {
			logrus.Error(err)
		} else {
			req.ContentLength = int64(len(b))
			if resp, err := http.DefaultClient.Do(req); err != nil {
				logrus.Error(err)
			} else {
				defer resp.Body.Close()
				ioutil.ReadAll(resp.Body)
			}
		}
	}()

	if strings.HasPrefix(request.PhysicalResourceId, "failed/") {
		return
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err == nil {
		if f, ok := customResourceTypes[request.ResourceType]; ok {
			success, err = f(request, cfg)
		} else {
			err = fmt.Errorf("Invalid custom resource type.")
		}
	}
}

func main() {
	if !terminal.IsTerminal(unix.Stdout) {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	if err := flags.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			// Exit with no error if --help was given. This is used to test the build.
			os.Exit(0)
		}
		logrus.Fatal(err)
	}

	lambda.Start(Handler)
}
