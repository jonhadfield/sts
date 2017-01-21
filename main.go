package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/urfave/cli"
	"os"
	"sort"
	"syscall"
	"time"
)

type cliArgs struct {
	displayOnly bool
}

func getMFASerial(sess client.ConfigProvider) (serialNo string) {
	// Try to load the MFA device serial using long term credentials
	svc := iam.New(sess)
	params := &iam.ListMFADevicesInput{}
	resp, err := svc.ListMFADevices(params)
	if err == nil {
		return *resp.MFADevices[0].SerialNumber

	}
	// Try to load the MFA device serial from environment variable
	envMFADevice := os.Getenv("AWS_MFA_DEVICE")
	if envMFADevice == "" {
		os.Getenv("MFA_DEVICE")
	}
	if envMFADevice == "" {
		fmt.Print("Enter serial number: ")
		fmt.Scanln(&serialNo)
	} else {
		serialNo = envMFADevice
	}
	return serialNo
}

func getSessionToken(sess client.ConfigProvider, hide bool, shell bool) {
	// fmt.Println("Getting MFA Serial...")
	serialNo := getMFASerial(sess)
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

	if hide != true {
		showCreds(*resp.Credentials.AccessKeyId, *resp.Credentials.SecretAccessKey,
			*resp.Credentials.SessionToken, *resp.Credentials.Expiration)
	}
	if shell == true {
		forkShell(*resp.Credentials.AccessKeyId, *resp.Credentials.SecretAccessKey,
			*resp.Credentials.SessionToken, *resp.Credentials.Expiration)
	}
}

func assumeRole(sess client.ConfigProvider, roleArn string, roleSessionName string, hide bool, shell bool) {
	// fmt.Println("Getting MFA Serial...")
	serialNo := getMFASerial(sess)
	var tokenVal string
	fmt.Print("Enter token value: ")
	fmt.Scanln(&tokenVal)
	svc := sts.New(sess)
	params := &sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: &roleSessionName,
		DurationSeconds: aws.Int64(3600),
		SerialNumber:    &serialNo,
		TokenCode:       &tokenVal,
	}
	resp, err := svc.AssumeRole(params)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if hide != true {
		showCreds(*resp.Credentials.AccessKeyId, *resp.Credentials.SecretAccessKey,
			*resp.Credentials.SessionToken, *resp.Credentials.Expiration)
	}
	if shell == true {
		forkShell(*resp.Credentials.AccessKeyId, *resp.Credentials.SecretAccessKey,
			*resp.Credentials.SessionToken, *resp.Credentials.Expiration)
	}
}

func showCreds(keyId string, secret string, sessionToken string, expiration time.Time) {
	fmt.Println("\n===========")
	fmt.Println("CREDENTIALS")
	fmt.Println("===========")
	fmt.Printf("AccessKeyId: %s\n", keyId)
	fmt.Printf("SecretAccessKey: %s\n", secret)
	fmt.Printf("SessionToken: %s\n", sessionToken)
	fmt.Printf("Expiration: %s\n", expiration)
}

func forkShell(keyId string, secret string, sessionToken string, expiration time.Time) {
	// Set environment variables and fork
	fmt.Println("\nLaunching new shell with temporary credentials...")
	os.Setenv("AWS_ACCESS_KEY_ID", keyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secret)
	os.Setenv("AWS_SECURITY_TOKEN", sessionToken)
	syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
}

func main() {
	app := cli.NewApp()
	app.Name = "sts"
	app.Version = "0.0.3"
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Jon Hadfield",
			Email: "jon@lessknown.co.uk",
		},
	}
	app.HelpName = "-"
	app.Usage = "Security Token Service"
	app.Description = ""

	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}

	app.Commands = []cli.Command{
		{
			Name:    "assume-role",
			Aliases: []string{"st"},
			Usage:   "get a session token",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "role-arn",
					Usage: "arn of the role being assumed",
				},
				cli.StringFlag{
					Name:  "role-session-name",
					Usage: "arn of the role being assumed",
				},
				cli.BoolFlag{
					Name:  "hide",
					Usage: "hide credentials",
				},
				cli.BoolFlag{
					Name:  "shell, s",
					Usage: "Fork to a shell with credentials set in environment",
				},
			},
			Action: func(c *cli.Context) error {
				roleArn := c.String("role-arn")
				roleSessionName := c.String("role-session-name")
				if roleArn == "" && roleSessionName == "" {
					return cli.NewExitError("error: --role-arn and --role-session-name must be specified", 1)
				} else if roleArn == "" {
					return cli.NewExitError("error: --role-arn must be specified", 1)
				} else if roleSessionName == "" {
					return cli.NewExitError("error: --role-session-name must be specified", 1)
				}
				assumeRole(sess, roleArn, roleSessionName, c.Bool("hide"), c.Bool("shell"))
				return nil
			},
		},
		{
			Name:    "get-session-token",
			Aliases: []string{"st"},
			Usage:   "get a session token",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "hide",
					Usage: "hide credentials",
				},
				cli.BoolFlag{
					Name:  "shell, s",
					Usage: "Fork to a shell with credentials set in environment",
				},
			},
			Action: func(c *cli.Context) error {
				getSessionToken(sess, c.Bool("hide"), c.Bool("shell"))
				return nil
			},
		},
		{
			Name:    "get-federation-token",
			Aliases: []string{"ft"},
			Usage:   "get a federation token",
			Action: func(c *cli.Context) error {
				fmt.Println("Not implemented")
				return nil
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)

}
