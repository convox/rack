package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/kinesis"
)

func SubscribeLogs(app string, output chan []byte, quit chan bool) error {
	resources, err := ListResources(app)

	if err != nil {
		return err
	}

	processes, err := ListProcesses(app)

	if err != nil {
		return err
	}

	done := make([](chan bool), len(processes))

	for i, ps := range processes {
		done[i] = make(chan bool)
		go subscribeKinesis(ps.Name, resources[fmt.Sprintf("%sKinesis", upperName(ps.Name))].PhysicalId, output, done[i])
	}

	return nil
}

func subscribeKinesis(prefix, stream string, output chan []byte, quit chan bool) {
	sreq := &kinesis.DescribeStreamInput{
		StreamName: aws.String(stream),
	}
	sres, err := Kinesis.DescribeStream(sreq)

	if err != nil {
		fmt.Printf("err1 %+v\n", err)
		// panic(err)
		return
	}

	shards := make([]string, len(sres.StreamDescription.Shards))

	for i, s := range sres.StreamDescription.Shards {
		shards[i] = *s.ShardID
	}

	done := make([](chan bool), len(shards))

	for i, shard := range shards {
		done[i] = make(chan bool)
		go subscribeKinesisShard(prefix, stream, shard, output, done[i])
	}
}

func subscribeKinesisShard(prefix, stream, shard string, output chan []byte, quit chan bool) {
	ireq := &kinesis.GetShardIteratorInput{
		ShardID:           aws.String(shard),
		ShardIteratorType: aws.String("LATEST"),
		StreamName:        aws.String(stream),
	}
	ires, err := Kinesis.GetShardIterator(ireq)

	if err != nil {
		fmt.Printf("err2 %+v\n", err)
		// panic(err)
		return
	}

	iter := *ires.ShardIterator

	for {
		select {
		case <-quit:
			fmt.Println("quitting")
			return
		default:
			greq := &kinesis.GetRecordsInput{
				ShardIterator: aws.String(iter),
			}
			gres, err := Kinesis.GetRecords(greq)

			if err != nil {
				fmt.Printf("err3 %+v\n", err)
				// panic(err)
				return
			}

			iter = *gres.NextShardIterator

			for _, record := range gres.Records {
				output <- []byte(fmt.Sprintf("%s: %s\n", prefix, string(record.Data)))
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}
