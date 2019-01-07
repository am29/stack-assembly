package stackassembly

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

type ChangeSetHandle struct {
	ID        string
	Changes   []Change
	IsUpdate  bool
	stackName string
	cf        cloudformationiface.CloudFormationAPI
}

func (csh ChangeSetHandle) Exec() error {
	_, err := csh.cf.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(csh.ID),
	})
	if err != nil {
		return err
	}

	stackInput := cloudformation.DescribeStacksInput{
		StackName: aws.String(csh.stackName),
	}

	ctx := aws.BackgroundContext()

	if csh.IsUpdate {
		return csh.cf.WaitUntilStackUpdateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
			w.MaxAttempts = 900
			w.Delay = request.ConstantWaiterDelay(2 * time.Second)
		})
	}

	return csh.cf.WaitUntilStackCreateCompleteWithContext(ctx, &stackInput, func(w *request.Waiter) {
		w.MaxAttempts = 900
		w.Delay = request.ConstantWaiterDelay(2 * time.Second)
	})
}
