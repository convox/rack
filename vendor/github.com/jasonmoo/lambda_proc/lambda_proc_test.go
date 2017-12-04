package lambda_proc

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestRunStream(t *testing.T) {

	const TestRecords = 100

	type Record struct {
		Id int `json:"id"`
	}

	ctx := &Context{
		AwsRequestID:             "awsRequestId",
		FunctionName:             "functionName",
		FunctionVersion:          "functionVersion",
		Invokeid:                 "invokeid",
		IsDefaultFunctionVersion: true,
		LogGroupName:             "logGroupName",
		LogStreamName:            "logStreamName",
		MemoryLimitInMB:          "memoryLimitInMB",
	}

	records := &bytes.Buffer{}
	enc := json.NewEncoder(records)
	for i := 0; i < TestRecords; i++ {
		data, err := json.Marshal(&Record{Id: i})
		if err != nil {
			t.Error(err)
		}
		if err := enc.Encode(&Payload{Context: ctx, Event: json.RawMessage(data)}); err != nil {
			t.Error(err)
		}
	}

	r, w := io.Pipe()

	go func() {
		RunStream(func(c *Context, data json.RawMessage) (interface{}, error) {
			if c.AwsRequestID != ctx.AwsRequestID {
				t.Errorf("Expected %v, got %v", ctx.AwsRequestID, c.AwsRequestID)
			}
			if c.FunctionName != ctx.FunctionName {
				t.Errorf("Expected %v, got %v", ctx.FunctionName, c.FunctionName)
			}
			if c.FunctionVersion != ctx.FunctionVersion {
				t.Errorf("Expected %v, got %v", ctx.FunctionVersion, c.FunctionVersion)
			}
			if c.Invokeid != ctx.Invokeid {
				t.Errorf("Expected %v, got %v", ctx.Invokeid, c.Invokeid)
			}
			if c.IsDefaultFunctionVersion != ctx.IsDefaultFunctionVersion {
				t.Errorf("Expected %v, got %v", ctx.IsDefaultFunctionVersion, c.IsDefaultFunctionVersion)
			}
			if c.LogGroupName != ctx.LogGroupName {
				t.Errorf("Expected %v, got %v", ctx.LogGroupName, c.LogGroupName)
			}
			if c.LogStreamName != ctx.LogStreamName {
				t.Errorf("Expected %v, got %v", ctx.LogStreamName, c.LogStreamName)
			}
			if c.MemoryLimitInMB != ctx.MemoryLimitInMB {
				t.Errorf("Expected %v, got %v", ctx.MemoryLimitInMB, c.MemoryLimitInMB)
			}
			var rec Record
			if err := json.Unmarshal(data, &rec); err != nil {
				t.Error(err)
			}
			return &rec, nil
		}, records, w)
	}()

	dec := json.NewDecoder(r)
	for i := 0; i < TestRecords; i++ {
		var resp Response
		if err := dec.Decode(&resp); err != nil {
			t.Error(err)
		}
		if resp.Error != nil {
			t.Errorf("Expected nil error, got: %v", resp.Error)
		}
		if resp.RequestId != i {
			t.Errorf("Expected %d, got %d", i, resp.RequestId)
		}
		data, ok := resp.Data.(map[string]interface{})
		if !ok {
			t.Errorf("Expected type map[string]interface{}, got %T", resp.Data)
		}
		if data["id"].(float64) != float64(i) {
			t.Errorf("Expected %d, got %v", i, data["id"])
		}
	}

}
