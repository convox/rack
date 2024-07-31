package handler

import (
	"fmt"
	"strconv"
	"time"
)

func HandleMathMax(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING Max")
		fmt.Printf("req %+v\n", req)
		return CreateMathMax(req)
	case "Update":
		fmt.Println("UPDATING Max")
		fmt.Printf("req %+v\n", req)
		return UpdateMathMax(req)
	case "Delete":
		fmt.Println("no need to delete")
		fmt.Printf("req %+v\n", req)
		return req.PhysicalResourceId, nil, nil
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func HandleMathMin(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING Min")
		fmt.Printf("req %+v\n", req)
		return CreateMathMin(req)
	case "Update":
		fmt.Println("UPDATING Min")
		fmt.Printf("req %+v\n", req)
		return UpdateMathMin(req)
	case "Delete":
		fmt.Println("no need to delete")
		fmt.Printf("req %+v\n", req)
		return req.PhysicalResourceId, nil, nil
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func CreateMathMax(req Request) (string, map[string]string, error) {
	val, err := mathMax(req)
	if err != nil {
		return "invalid", nil, err
	}

	return fmt.Sprintf("mathmax-%d", time.Now().UnixNano()), map[string]string{
		"Value": val,
	}, nil
}

func UpdateMathMax(req Request) (string, map[string]string, error) {
	val, err := mathMax(req)
	if err != nil {
		return "invalid", nil, err
	}

	return req.PhysicalResourceId, map[string]string{
		"Value": val,
	}, nil
}

func CreateMathMin(req Request) (string, map[string]string, error) {
	val, err := mathMin(req)
	if err != nil {
		return "invalid", nil, err
	}

	return fmt.Sprintf("mathmin-%d", time.Now().UnixNano()), map[string]string{
		"Value": val,
	}, nil
}

func UpdateMathMin(req Request) (string, map[string]string, error) {
	val, err := mathMin(req)
	if err != nil {
		return "invalid", nil, err
	}

	return req.PhysicalResourceId, map[string]string{
		"Value": val,
	}, nil
}

func mathMax(req Request) (string, error) {
	x, y, err := parseXY(req)
	if err != nil {
		return "", err
	}

	if y > x {
		x = y
	}

	return strconv.Itoa(x), nil
}

func mathMin(req Request) (string, error) {
	x, y, err := parseXY(req)
	if err != nil {
		return "", err
	}

	if y < x {
		x = y
	}

	return strconv.Itoa(x), nil
}

func parseXY(req Request) (int, int, error) {
	xStr := req.ResourceProperties["X"].(string)
	yStr := req.ResourceProperties["Y"].(string)

	x, err := strconv.Atoi(xStr)
	if err != nil {
		return 0, 0, err
	}

	y, err := strconv.Atoi(yStr)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}
