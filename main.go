package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"os"
	"syscall"
)

func main() {
	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}

	// Try to load the MFA device serial from environment variable
	envMFADevice := os.Getenv("AWS_MFA_DEVICE")
	var serialNo string
	if envMFADevice == "" {
		fmt.Print("Enter serial number: ")
		fmt.Scanln(&serialNo)
	} else {
		serialNo = envMFADevice
	}
	var tokenVal string
	fmt.Print("Enter token value: ")
	fmt.Scanln(&tokenVal)
	svc := sts.New(sess)

	params := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(3600),
		SerialNumber:    &serialNo,
		TokenCode:       &tokenVal,
	}
	resp, err := svc.GetSessionToken(params)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("Credentials expire: %s\n", *resp.Credentials.Expiration)

	// Set EnvVars
	os.Setenv("AWS_ACCESS_KEY_ID", *resp.Credentials.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", *resp.Credentials.SecretAccessKey)
	os.Setenv("AWS_SECURITY_TOKEN", *resp.Credentials.SessionToken)
	syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}
