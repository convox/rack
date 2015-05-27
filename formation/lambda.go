package formation

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/lambda"
)

func HandleLambdaFunction(req Request) (string, error) {
	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING LAMBDA")
		fmt.Printf("req %+v\n", req)
		return LambdaFunctionCreate(req)
	case "Update":
		fmt.Println("UPDATING LAMBDA")
		fmt.Printf("req %+v\n", req)
		return LambdaFunctionUpdate(req)
	case "Delete":
		fmt.Println("DELETING LAMBDA")
		fmt.Printf("req %+v\n", req)
		return LambdaFunctionDelete(req)
	}

	return "", fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func LambdaFunctionCreate(req Request) (string, error) {
	bres, err := http.Get(req.ResourceProperties["Zip"].(string))

	if err != nil {
		return "", err
	}

	defer bres.Body.Close()

	body, err := ioutil.ReadAll(bres.Body)

	// zip := make([]byte, base64.StdEncoding.EncodedLen(len(body)))

	// base64.StdEncoding.Encode(zip, body)

	// fmt.Printf("len(zip) %+v\n", len(zip))

	memory := 128
	timeout := 5

	if m, ok := req.ResourceProperties["Memory"]; ok {
		memory, _ = strconv.Atoi(m.(string))
	}

	if t, ok := req.ResourceProperties["Timeout"]; ok {
		timeout, _ = strconv.Atoi(t.(string))
	}

	role := fmt.Sprintf("arn:aws:iam::%s:role/%s", req.ResourceProperties["AccountId"].(string), req.ResourceProperties["Role"].(string))

	res, err := Lambda().CreateFunction(&lambda.CreateFunctionInput{
		FunctionName: aws.String(req.ResourceProperties["Name"].(string)),
		Handler:      aws.String(req.ResourceProperties["Handler"].(string)),
		MemorySize:   aws.Long(int64(memory)),
		Timeout:      aws.Long(int64(timeout)),
		Role:         aws.String(role),
		Runtime:      aws.String(req.ResourceProperties["Runtime"].(string)),
		Code: &lambda.FunctionCode{
			ZipFile: body,
		},
	})

	if err != nil {
		return "", err
	}

	return *res.FunctionARN, nil
}

func LambdaFunctionUpdate(req Request) (string, error) {
	fmt.Printf("req %+v\n", req)
	return "", fmt.Errorf("could not update")
}

func LambdaFunctionDelete(req Request) (string, error) {
	_, err := Lambda().DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: aws.String(req.PhysicalResourceId),
	})

	if err != nil {
		return "", err
	}

	return "", nil
}
