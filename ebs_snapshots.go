//
// PURPOSE: find instances with backup=yes|true tags and snapshot the volumes
//

package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/session"
	"github.com/awslabs/aws-sdk-go/service/ec2"

	"fmt"
)

func Handler() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	client := ec2.New(sess)

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:backup"),
				Values: []*string{
					aws.String("true"), aws.String("True"), aws.String("TRUE"),
					aws.String("yes"), aws.String("Yes"), aws.String("YES"),
				},
			},
		},
	}

	result, err := client.DescribeInstances(params)
	if err != nil {
		fmt.Println("[ERROR] ", err)
	}

	for ridx, _ := range result.Reservations {
		for iidx, inst := range result.Reservations[ridx].Instances {
			name := "None"
			for _, keys := range inst.Tags {
				if *keys.Key == "Name" {
					name = *keys.Value
					for _, dev := range result.Reservations[ridx].Instances[iidx].BlockDeviceMappings {
						var volume_id = *dev.Ebs.VolumeId
						fmt.Printf("{ name: %s, volume_id: %s }\n", name, volume_id)
						input := &ec2.CreateSnapshotInput{
							Description: aws.String("lambda.ebs_snapshots - " + name + " - " + volume_id),
							VolumeId:    aws.String(volume_id),
						}
						_, err := client.CreateSnapshot(input)
						if err != nil {
							fmt.Println(err.Error())
						}
					}
				}
			}
		}
	}
}

func main() {
	lambda.Start(Handler)
}
