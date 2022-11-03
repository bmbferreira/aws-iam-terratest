package test

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gruntwork-io/terratest/modules/retry"

	"os"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/files"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	test_structure "github.com/gruntwork-io/terratest/modules/test-structure"
)

var currentIP = GetCurrentIP()

// An example of how to test a Terraform module that contains a user with an iam policy that restricts actions by ip address using Terratest.
func TestTerraformIAMIpsExample(t *testing.T) {
	t.Parallel()

	tmpFolder := test_structure.CopyTerraformFolderToTemp(t, "../", "modules/iam-user")
	// Installl the provider.tf with defered deletion
	defer os.Remove(tmpFolder + "/provider.tf")
	files.CopyFile("provider.tf", tmpFolder+"/provider.tf")
	// At the end of the test, run `terraform destroy` to clean up any resources that were created
	defer test_structure.RunTestStage(t, "teardown", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, tmpFolder)
		terraform.Destroy(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "setup ip address different from current one as allowed", func() {
		terraformOptions := configureTerraformOptions(t, tmpFolder, []string{"1.2.3.4"})

		// Save the options so later test stages can use them
		test_structure.SaveTerraformOptions(t, tmpFolder, terraformOptions)

		// This will run `terraform init` and `terraform apply` and fail the test if there are any errors
		terraform.InitAndApply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate we can't access sqs with current IP", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, tmpFolder)
		result, err := callSQSListQueues(t, terraformOptions)

		// Since current IP is not allowed, we should get an error and nothing on the result
		assert.NotNil(t, err)
		assert.Nil(t, result.QueueUrls)
	})

	test_structure.RunTestStage(t, "setup current ip address as allowed", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, tmpFolder)
		terraformOptions.Vars["allowed_ips"] = []string{currentIP}

		// Save the options so later test stages can use them
		test_structure.SaveTerraformOptions(t, tmpFolder, terraformOptions)

		// This will run `terraform init` and `terraform apply` and fail the test if there are any errors
		terraform.InitAndApply(t, terraformOptions)
	})

	test_structure.RunTestStage(t, "validate we can access SQS with current IP", func() {
		terraformOptions := test_structure.LoadTerraformOptions(t, tmpFolder)
		result, err := callSQSListQueues(t, terraformOptions)

		// Since current IP is now allowed, we should get no error and something as a result
		assert.NotNil(t, result.QueueUrls)
		assert.Nil(t, err)
	})
}

func configureTerraformOptions(t *testing.T, tmpFolder string, allowed_ips []string) *terraform.Options {
	// A unique ID we can use to namespace resources so we don't clash with anything already in the AWS account or
	// tests running in parallel
	uniqueID := random.UniqueId()

	userName := fmt.Sprintf("user-%s", uniqueID)
	iamPolicyName := fmt.Sprintf("iprestricted-%s", uniqueID)

	// Construct the terraform options with default retryable errors to handle the most common retryable errors in
	// terraform testing.
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: tmpFolder,

		// Variables to pass to our Terraform code using -var options
		Vars: map[string]interface{}{
			"user_name":       userName,
			"iam_policy_name": iamPolicyName,
			"allowed_ips":     allowed_ips,
		},
	})

	return terraformOptions
}

// Calls SQS ListQueues using IAM user credentials from terraform output
func callSQSListQueues(t *testing.T, terraformOptions *terraform.Options) (*sqs.ListQueuesOutput, error) {
	// Run `terraform output` to get the value of an output variable
	accessId := terraform.Output(t, terraformOptions, "aws_iam_access_key_id")
	accessSecret := terraform.Output(t, terraformOptions, "aws_iam_access_key_secret")
	session, err := aws.CreateAwsSessionWithCreds("us-east-1", accessId, accessSecret)
	if err != nil {
		log.Printf("Error creating session: %v", err)
	}
	// Create a SQS client from just a session.
	svc := sqs.New(session)
	// It can take a few seconds for the iam user keys starting to work
	result, err := retry.DoWithRetryInterfaceE(t, fmt.Sprintf("Calling SQS from host %s", currentIP), 3, 5*time.Second, func() (interface{}, error) {

		result, err := svc.ListQueues(nil)
		if err != nil {
			log.Printf("Error listing queues: %v", err)
		}
		return result, err
	})
	return result.(*sqs.ListQueuesOutput), err
}

// Gets current public IP
func GetCurrentIP() string {
	url := "https://api.ipify.org?format=text"
	log.Println("Getting IP address from ipify...")
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Current public IP is: %s", ip)
	return string(ip[:])
}
