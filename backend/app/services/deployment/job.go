package deployment

import "errors"

// DeployApplicationJob runs one deployment off the queue.
type DeployApplicationJob struct{}

func (j *DeployApplicationJob) Signature() string {
	return "ebox:deploy-application"
}

func (j *DeployApplicationJob) Handle(args ...any) error {
	if len(args) == 0 {
		return errors.New("deployment: the job is missing the deployment id")
	}

	deploymentUUID, ok := args[0].(string)
	if !ok || deploymentUUID == "" {
		return errors.New("deployment: the job received an invalid deployment id")
	}

	return Default().Run(deploymentUUID)
}
