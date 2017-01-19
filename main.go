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

func getMFASerial() (serialNo string) {
	// Try to load the MFA device serial using long term credentials

	// Try to load the MFA device serial from environment variable
	envMFADevice := os.Getenv("AWS_MFA_DEVICE")
	if envMFADevice == "" {
		os.Getenv("MFA_DEVICE")
	}
	//var serialNo string
	if envMFADevice == "" {
		fmt.Print("Enter serial number: ")
		fmt.Scanln(&serialNo)
	} else {
		serialNo = envMFADevice
	}
	return serialNo
}

func getSessionToken(display bool, shell bool) {
	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}

	var tokenVal string
	fmt.Print("Enter token value: ")
	fmt.Scanln(&tokenVal)
	svc := sts.New(sess)
	serialNo := getMFASerial()
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

	fmt.Println("\n===========")
	fmt.Println("\nCREDENTIALS")
	fmt.Println("\n===========")
	fmt.Printf("AccessKeyId: %s\n", *resp.Credentials.AccessKeyId)
	fmt.Printf("SecretAccessKey: %s\n", *resp.Credentials.SecretAccessKey)
	fmt.Printf("SessionToken: %s\n", *resp.Credentials.SessionToken)
	fmt.Printf("Expiration: %s\n", *resp.Credentials.Expiration)

	if shell == true {
		fmt.Println("Launching new shell with temporary credentials...")
		syscall.Exec(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, syscall.Environ())
	}
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
	//app.Flags = []cli.Flag{
	//	cli.StringFlag{
	//		Name:  "display, d",
	//		Usage: "Display credentials only",
	//	},
	//	cli.BoolTFlag{
	//		Name:  "shell, s",
	//		Usage: "Fork to a shell with credentials set in environment",
	//	},
	//}

	app.Commands = []cli.Command{
		{
			Name:    "get-session-token",
			Aliases: []string{"st"},
			Usage:   "get a session token",
			Flags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "hide, d",
					Usage: "hide credentials",
				},
				cli.BoolFlag{
					Name:  "shell, s",
					Usage: "Fork to a shell with credentials set in environment",
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println(app.Flags)
				fmt.Println(c.Args())
				fmt.Println(c.Bool("display"))
				getSessionToken(c.Bool("display"), c.Bool("shell"))
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
