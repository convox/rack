package handler

import "fmt"

func HandleSNSSubcription(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING SNSSUBSCRIPTION")
		fmt.Printf("req %+v\n", req)
		return SNSSubscriptionCreate(req)
	case "Update":
		fmt.Println("UPDATING SNSSUBSCRIPTION")
		fmt.Printf("req %+v\n", req)
		return SNSSubscriptionUpdate(req)
	case "Delete":
		fmt.Println("DELETING SNSSUBSCRIPTION")
		fmt.Printf("req %+v\n", req)
		return SNSSubscriptionDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func SNSSubscriptionCreate(req Request) (string, map[string]string, error) {
	// nop
	return req.PhysicalResourceId, nil, nil
}

func SNSSubscriptionUpdate(req Request) (string, map[string]string, error) {
	// nop
	return req.PhysicalResourceId, nil, nil
}

func SNSSubscriptionDelete(req Request) (string, map[string]string, error) {
	// nop
	return req.PhysicalResourceId, nil, nil
}
