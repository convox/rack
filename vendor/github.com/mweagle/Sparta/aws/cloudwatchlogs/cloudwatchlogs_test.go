package cloudwatchlogs

import (
	"encoding/json"
	"testing"
)

const logMessageTestData = `
{
  "awslogs":
  {
  "data": "H4sIAAAAAAAAAK2TW2/TQBCF/8rK4jFOZu+7fnOVUHFJQbEBiTqq1va6smTHwXbShqr/nUlTBEiAWoH27ZzR2W+OvXdB64fBXfv0sPVBFMzjNL5aLpIkPl8Ek6C72fgeZWCaSitAGi5Qbrrr877bbdGZuZth1rg2L91s7/uh7jbDaSIZe+9aHGFA1QzYjMHs8sXbOF0k6Vr6KldVwU3FqKiMNUwzarjSnrKc6xwjhl0+FH29HTHyZd2MGB5El0Gydf3o4u22qQt39K6Wh5NdUCcKK50Q0ueVBOFYSZ21FZNeVRUvhbTe2SpYP/At9n4zHiPvgrpETM6EMhIscKMRWRoQRnIhNafKgBaaa6zBMqMQVwvFKSipABB1rLHF0bVYCBVSWm6YtVyqyfd2MT5J41VKVv7LDkdflRGxqrLGKR6W2kBIqZehy7UJlcXLfc7Kwufk46nSiDwWl22C+8lvgI0GBcwIyrFsCVwLBBCCYq9CKS6stFRYAGWE/TOw/Rn4+NlCYCGDFHhETUTtlEvzORufQp6NF13pXydkP93DlMKUqwmJPyUkmb9BjeHh7N+3kX+p/5dtFhfz55b/H+joE+lWi/fvnv93ZON81z+8gYjAVDLSDtl4VjeNL8kPhwKgQbJx6duuP5Ck/upRZYYsz1B0t+TR+DB4vJjyB/24/Pr+G81LpuMfBAAA"
  }
}
`

func TestUnmarshal(t *testing.T) {
	var event Event
	err := json.Unmarshal([]byte(logMessageTestData), &event)
	if nil != err {
		t.Errorf("Failed to unmarshal log event message")
	}
	data, err := event.AWSLogs.DecodedData()
	if nil != err {
		t.Errorf("Failed to decode event data: " + err.Error())
	}
	if len(data.LogEvents) != 4 {
		t.Errorf("Failed to unmarshal 4 LogEvent entries")
	}
}
