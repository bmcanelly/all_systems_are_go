//
// PURPOSE: find instances with a Name:tag and lambda_snapshot=true tag; snapshot their ebs volumes
//

package main

import (
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/session"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	logrus "github.com/sirupsen/logrus"
)

type Snapshot struct {
	VolumeId    string
	Description string
}

type SnapshotInput struct {
	Snapshots []Snapshot
}

func Handler() {
	var wg sync.WaitGroup
	var ssi SnapshotInput

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.InfoLevel)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	client := ec2.New(sess)

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:lambda_snapshot"),
				Values: []*string{
					aws.String("true"),
				},
			},
		},
	}
	instances, err := client.DescribeInstances(params)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Instances found: ", len(instances.Reservations))

	for r, _ := range instances.Reservations {
		wg.Add(1)
		go func(r int) {
			defer wg.Done()
			for i, inst := range instances.Reservations[r].Instances {
				for _, keys := range inst.Tags {
					if *keys.Key == "Name" {
						for _, dev := range instances.Reservations[r].Instances[i].BlockDeviceMappings {
							logrus.Info("name: ", *keys.Value, "  volume_id: ", *dev.Ebs.VolumeId)
							ssi.Snapshots = append(ssi.Snapshots, Snapshot{
								VolumeId:    *dev.Ebs.VolumeId,
								Description: "lambda.create_ebs_snapshots - " + *keys.Value + " - " + *dev.Ebs.VolumeId,
							})
						}
					}
				}
			}
		}(r)
	}
	wg.Wait()

	sumSnapshots := len(ssi.Snapshots)
	logrus.Info("Snapshots found: ", sumSnapshots)

	for i := 0; i < sumSnapshots; i++ {
		snapshot := ssi.Snapshots[i]

		wg.Add(1)
		go func(s Snapshot) {
			defer wg.Done()
			logrus.Info(s.Description)
			input := ec2.CreateSnapshotInput{
				Description: aws.String(s.Description),
				VolumeId:    aws.String(s.VolumeId),
			}
			_, err := client.CreateSnapshot(&input)
			if err != nil {
				logrus.Error(err)
			}
		}(snapshot)
	}
	wg.Wait()

	logrus.Info("done")
}

func main() {
	lambda.Start(Handler)
}
