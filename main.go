package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var qURL = "https://sqs.us-east-1.amazonaws.com/776913033148/moments.fifo"
var awsSession *session.Session

type Request struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type Response struct {
	Message string `json:"message"`
	Ok      bool   `json:"ok"`
}

func momentJobCount() int {
	sqsService := sqs.New(awsSession)
	res, err := sqsService.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       &qURL,
		AttributeNames: aws.StringSlice([]string{"All"}),
	})
	check(err, "Problem getting moment queue attributes")
	numMsgs, err := strconv.Atoi(*res.Attributes["ApproximateNumberOfMessages"])
	check(err, "Problem converting number of moment queue messages to integer")

	return numMsgs
}

func momentProcessorRunning() bool {
	ecsService := ecs.New(awsSession)
	res, err := ecsService.ListTasks(&ecs.ListTasksInput{
		Family:        aws.String("mfe-moment-processor"),
		DesiredStatus: aws.String("RUNNING"),
	})
	check(err, "Problem fetching a list of running moment processor tasks")
	return len(res.TaskArns) > 0
}

func startMomentProcessor() (*ecs.RunTaskOutput, error) {
	ecsService := ecs.New(awsSession)
	return ecsService.RunTask(&ecs.RunTaskInput{
		TaskDefinition: aws.String("mfe-moment-processor"),
		LaunchType:     aws.String(ecs.LaunchTypeFargate),
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: aws.String(ecs.AssignPublicIpEnabled),
				Subnets:        aws.StringSlice([]string{"subnet-2aad8661"}),
			},
		},
	})
}

func Handler(request Request) (Response, error) {

	awsSession, _ = session.NewSession(&aws.Config{Region: aws.String("us-east-1")})

	// 1) query sqs queue for any available messages
	mjc := momentJobCount()
	fmt.Println("Moment job count", mjc)

	// 2) query ecs tasks for moment-processor to see if any are running, pending, provisioning
	mpc := momentProcessorRunning()
	fmt.Println("Moment job processing", mpc)

	// 3) if messages are available and no moment-processor is running, start task
	if mjc > 0 && !mpc {
		fmt.Println("Moment jobs to process, but no processor is running. Starting now...")
		res, err := startMomentProcessor()
		check(err, "Problem starting moment processor")
		fmt.Println(res)
	} else {
		fmt.Println("Moment processor not needed at this time")
	}

	return Response{
		Message: fmt.Sprintf("MomentScheduler Complete"),
		Ok:      true,
	}, nil
}

func main() {
	lambda.Start(Handler)
}

func check(err error, msg string) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
