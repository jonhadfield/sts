package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

func getSessionToken() {
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
	app.Copyright = "(c) 2017 Jon Hadfield"
	app.HelpName = "-"
	app.Usage = "Security Token Service"
	app.UsageText = "contrive - demonstrating the available API"
	app.ArgsUsage = "[args and such]"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "lang, l",
			Value: "english",
			Usage: "Language for the greeting",
		},
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Load configuration from `FILE`",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "get-session-token",
			Aliases: []string{"st"},
			Usage:   "get a session token",
			Action: func(c *cli.Context) error {
				getSessionToken()
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
