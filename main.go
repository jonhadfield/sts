package main

import (
	"fmt"
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

func getMFASerial(sess client.ConfigProvider) (serialNo string) {
	// Try to load the MFA device serial using long term credentials
	svc := iam.New(sess)
	params := &iam.ListMFADevicesInput{}
	resp, err := svc.ListMFADevices(params)
	if err == nil {
		return *resp.MFADevices[0].SerialNumber

	}
	// Try to load the MFA device serial from environment variables
	serialNo = os.Getenv("AWS_MFA_DEVICE")
	if serialNo == "" {
		serialNo = os.Getenv("MFA_DEVICE")
	}
	return serialNo
}

func getSessionToken(sess client.ConfigProvider, duration int64, serialNo string,
	tokenCode string, hide bool, shell bool) {
	if duration == 0 {
		duration = 43200
	}
	if serialNo == "" {
		serialNo = getMFASerial(sess)
	}
	// Request token code if serial number is set, but token code is not
	if serialNo != "" && tokenCode == "" {
		fmt.Print("Enter token value: ")
		fmt.Scanln(&tokenCode)
	}
	svc := sts.New(sess)
	params := &sts.GetSessionTokenInput{
		DurationSeconds: &duration,
	}
	if tokenCode != "" && serialNo != "" {
		params.SerialNumber = &serialNo
		params.TokenCode = &tokenCode
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

func getFederationToken(sess client.ConfigProvider, duration int64, name string,
	policy string, hide bool, shell bool) {

	svc := sts.New(sess)
	params := &sts.GetFederationTokenInput{
		DurationSeconds: &duration,
		Name:            &name,
		Policy:          &policy,
	}

	resp, err := svc.GetFederationToken(params)

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

func assumeRole(sess client.ConfigProvider, roleArn string, roleSessionName string, duration int64,
	serialNo string, tokenCode string, hide bool, shell bool) {
	if duration == 0 {
		duration = 3600
	}
	if serialNo == "" {
		serialNo = getMFASerial(sess)
	}
	// Request token code if serial number is set, but token code is not
	if serialNo != "" && tokenCode == "" {
		fmt.Print("Enter token value: ")
		fmt.Scanln(&tokenCode)
	}
	svc := sts.New(sess)
	params := &sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: &roleSessionName,
		DurationSeconds: &duration,
	}
	if tokenCode != "" && serialNo != "" {
		params.SerialNumber = &serialNo
		params.TokenCode = &tokenCode
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

func unsetAWSEnvvars() {
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_ACCESS_KEY")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SECRET_KEY")
	os.Unsetenv("AWS_SESSION_TOKEN")
}

func getSession() (sess *session.Session) {
	sess, err := session.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return sess
}

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true

	app.Name = "sts"
	app.Version = "0.1.10"
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

	app.Commands = []cli.Command{
		{
			Name:    "assume-role",
			Aliases: []string{"ar"},
			Usage:   "Return temporary credentials for an assumed role",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "role-arn",
					Usage: "Arn of the role being assumed",
				},
				cli.StringFlag{
					Name:  "role-session-name",
					Usage: "Arn of the role being assumed",
				},
				cli.Int64Flag{
					Name:  "duration-seconds",
					Usage: "Time for temporary credentials to remain valid",
					Value: 3600,
				},
				cli.StringFlag{
					Name:  "serial-number",
					Usage: "The MFA device identifier",
				},
				cli.StringFlag{
					Name:  "token-code",
					Usage: "The output generated by the MFA device",
				},
				cli.BoolFlag{
					Name:  "hide",
					Usage: "Hide credentials",
				},
				cli.BoolFlag{
					Name:  "shell, s",
					Usage: "Fork to a shell with credentials set in environment",
				},
				cli.BoolFlag{
					Name:  "unset-env, u",
					Usage: "Unset AWS environment variables before acquiring credentials",
				},
			},
			Action: func(c *cli.Context) error {
				roleArn := c.String("role-arn")
				roleSessionName := c.String("role-session-name")
				if roleArn == "" && roleSessionName == "" {
					return cli.NewExitError("error: "+
						"--role-arn and --role-session-name must be specified", 1)
				} else if roleArn == "" {
					return cli.NewExitError("error: "+
						"--role-arn must be specified", 1)
				} else if roleSessionName == "" {
					return cli.NewExitError("error: "+
						"--role-session-name must be specified", 1)
				}
				if c.Bool("unset-env") {
					unsetAWSEnvvars()
				}
				sess := getSession()

				assumeRole(sess, roleArn, roleSessionName, c.Int64("duration-seconds"),
					c.String("serial-number"), c.String("token-code"),
					c.Bool("hide"), c.Bool("shell"))
				return nil
			},
		},
		{
			Name:    "assume-role-with-saml",
			Aliases: []string{"arws"},
			Usage:   "Not yet implemented",
			Action: func(c *cli.Context) error {
				fmt.Println("Not implemented")
				return nil
			},
		},
		{
			Name:    "assume-role-with-web-identity",
			Aliases: []string{"arwwi"},
			Usage:   "Not yet implemented",
			Action: func(c *cli.Context) error {
				fmt.Println("Not implemented")
				return nil
			},
		},

		{
			Name:    "get-federation-token",
			Aliases: []string{"gft"},
			Usage:   "Return temporary credentials for a federated user",
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name:  "duration-seconds",
					Usage: "Time for temporary credentials to remain valid",
					Value: 43200,
				},
				cli.StringFlag{
					Name:  "name",
					Usage: "Name of the federated user",
				},
				cli.StringFlag{
					Name:  "policy",
					Usage: "IAM policy to scope down credentials",
				},
				cli.BoolFlag{
					Name:  "hide",
					Usage: "Hide credentials",
				},
				cli.BoolFlag{
					Name:  "shell, s",
					Usage: "Fork to a shell with credentials set in environment",
				},
				cli.BoolFlag{
					Name:  "unset-env, u",
					Usage: "Unset AWS environment variables before acquiring credentials",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Bool("unset-env") {
					unsetAWSEnvvars()
				}
				sess := getSession()
				getFederationToken(sess, c.Int64("duration-seconds"), c.String("name"),
					c.String("policy"), c.Bool("hide"), c.Bool("shell"))
				return nil
			},
		},
		{
			Name:    "get-session-token",
			Aliases: []string{"gst"},
			Usage:   "Return temporary credentials for a user",
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name:  "duration-seconds",
					Usage: "Time for temporary credentials to remain valid",
					Value: 43200,
				},
				cli.StringFlag{
					Name:  "serial-number",
					Usage: "The MFA device identifier",
				},
				cli.StringFlag{
					Name:  "token-code",
					Usage: "The output generated by the MFA device",
				},
				cli.BoolFlag{
					Name:  "hide",
					Usage: "Hide credentials",
				},
				cli.BoolFlag{
					Name:  "shell, s",
					Usage: "Fork to a shell with credentials set in environment",
				},
				cli.BoolFlag{
					Name:  "unset-env, u",
					Usage: "Unset AWS environment variables before acquiring credentials",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Bool("unset-env") {
					unsetAWSEnvvars()
				}
				sess := getSession()
				getSessionToken(sess, c.Int64("duration-seconds"), c.String("serial-number"),
					c.String("token-code"), c.Bool("hide"), c.Bool("shell"))
				return nil
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	app.Run(os.Args)

}
