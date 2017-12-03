package main

import (
	"encoding/json"

	"github.com/jasonmoo/lambda_proc"
)

func main() {

	lambda_proc.Run(func(context *lambda_proc.Context, eventJSON json.RawMessage) (interface{}, error) {

		var v map[string]interface{}
		if err := json.Unmarshal(eventJSON, &v); err != nil {
			return nil, err
		}
		return v, nil

	})

}
