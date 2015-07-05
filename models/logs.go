package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/kinesis"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/rds"
)

func subscribeKinesis(stream string, output chan []byte, quit chan bool) {
	sreq := &kinesis.DescribeStreamInput{
		StreamName: aws.String(stream),
	}
	sres, err := Kinesis().DescribeStream(sreq)

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
		go subscribeKinesisShard(stream, shard, output, done[i])
	}
}

func subscribeKinesisShard(stream, shard string, output chan []byte, quit chan bool) {
	ireq := &kinesis.GetShardIteratorInput{
		ShardID:           aws.String(shard),
		ShardIteratorType: aws.String("TRIM_HORIZON"),
		StreamName:        aws.String(stream),
	}

	ires, err := Kinesis().GetShardIterator(ireq)

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
			gres, err := Kinesis().GetRecords(greq)

			if err != nil {
				fmt.Printf("err3 %+v\n", err)
				// panic(err)
				return
			}

			iter = *gres.NextShardIterator

			for _, record := range gres.Records {
				output <- []byte(fmt.Sprintf("%s\n", string(record.Data)))
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

func subscribeRDS(prefix, id string, output chan []byte, quit chan bool) {
	// Get latest log file details via pagination tokens
	details := rds.DescribeDBLogFilesDetails{}
	marker := ""

	for {
		params := &rds.DescribeDBLogFilesInput{
			DBInstanceIdentifier: aws.String(id),
			MaxRecords:           aws.Long(100),
		}

		if marker != "" {
			params.Marker = aws.String(marker)
		}

		res, err := RDS().DescribeDBLogFiles(params)

		if err != nil {
			panic(err)
		}

		if res.Marker == nil {
			files := res.DescribeDBLogFiles
			details = *files[len(files)-1]

			break
		}

		marker = *res.Marker
	}

	// Get last 50 log lines
	params := &rds.DownloadDBLogFilePortionInput{
		DBInstanceIdentifier: aws.String(id),
		LogFileName:          aws.String(*details.LogFileName),
		NumberOfLines:        aws.Long(50),
	}

	res, err := RDS().DownloadDBLogFilePortion(params)

	if err != nil {
		panic(err)
	}

	output <- []byte(fmt.Sprintf("%s: %s\n", prefix, *res.LogFileData))

	params.Marker = aws.String(*res.Marker)

	for {
		select {
		case <-quit:
			fmt.Println("quitting")
			return
		default:
			res, err := RDS().DownloadDBLogFilePortion(params)

			if err != nil {
				panic(err)
			}

			if *params.Marker != *res.Marker {
				params.Marker = aws.String(*res.Marker)

				output <- []byte(fmt.Sprintf("%s: %s\n", prefix, *res.LogFileData))
			}

			time.Sleep(1000 * time.Millisecond)
		}
	}
}
