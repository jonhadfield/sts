package subtocheck

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"crypto/tls"
	"os"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/pkg/errors"
	"gopkg.in/gomail.v2"
)

func extractEmail(input string) (output string) {
	if strings.Contains(input, "<") {
		output = GetStringInBetween(input, "<", ">")
	} else {
		output = input
	}
	return
}

func emailConfigDefined(email Email) (result bool) {
	if !reflect.DeepEqual(email, Email{}) {
		result = true
	}
	return
}

func validateEmailSettings(email Email) (err error) {
	supportedProviders := []string{"ses", "smtp"}
	if emailConfigDefined(email) {
		if email.Provider == "" {
			err = fmt.Errorf("email provider not specified")
			return
		}

		if email.Source == "" {
			err = fmt.Errorf("email source not specified")
			return
		}

		if !StringInSlice(email.Provider, supportedProviders) {
			err = fmt.Errorf("email provider '%s' not supported", email.Provider)
			return
		}
		emailRegexp := regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
		// validate recipient email addresses
		for _, emailAddr := range email.Recipients {
			if !emailRegexp.MatchString(extractEmail(emailAddr)) {
				err = fmt.Errorf("invalid email address '%s'", extractEmail(emailAddr))
				return
			}
		}
		// validate source email address
		if !emailRegexp.MatchString(extractEmail(email.Source)) {
			err = fmt.Errorf("invalid email address '%s'", extractEmail(email.Source))
			return
		}
	}
	return
}

func generateDNSIssueList(dnsIssues []issue) (filePath string) {
	timeStamp := time.Now().UTC().Format("20060102150405")
	filePath = fmt.Sprintf("dns_issues_%s.txt", timeStamp)
	// convert issues to file content
	var buffer bytes.Buffer
	for _, dnsIssue := range dnsIssues {
		buffer.WriteString(dnsIssue.fqdn + " - " + dnsIssue.err.Error() + "\n")
	}
	f, createFileErr := os.Create(filePath)
	if createFileErr != nil {
		panic(createFileErr)
	}
	defer f.Close()
	_, writeErr := f.Write(buffer.Bytes())
	if writeErr != nil {
		panic(writeErr)
	}
	f.Sync()
	return
}

func generateRequestIssueList(requestIssues []issue) (filePath string) {
	timeStamp := time.Now().UTC().Format("20060102150405")
	filePath = fmt.Sprintf("request_issues_%s.txt", timeStamp)
	// convert issues to file content
	var buffer bytes.Buffer
	for _, requestIssue := range requestIssues {
		buffer.WriteString(requestIssue.url + " - " + requestIssue.err.Error() + "\n")
	}
	f, createFileErr := os.Create(filePath)
	if createFileErr != nil {
		panic(createFileErr)
	}
	defer f.Close()
	_, writeErr := f.Write(buffer.Bytes())
	if writeErr != nil {
		panic(writeErr)
	}
	f.Sync()
	return
}

func emailResults(email Email, pIssues processedIssues) (err error) {
	msg := gomail.NewMessage()
	msg.SetHeader("From", email.Source)
	var emailSubject string
	if email.Subject != "" {
		emailSubject = email.Subject
	} else {
		emailSubject = "AWS Account Scan"
	}

	if len(pIssues.potVulns) > 0 {
		emailSubject += " - potential vulnerabilities found"
	} else {
		emailSubject += " - no potential vulnerabilities found"
	}
	msg.SetHeader("Subject", emailSubject)

	body := "<font face=\"Courier New, Courier, monospace\">" +
		"&nbsp;Issues<br/>" +
		"--------" +
		"<br/>" +
		"</font>" +
		"<table border=\"0\" cellpadding=\"3\" cellspacing=\"3\" width=\"300\">" +
		"<tr>" +
		"<td><font face=\"Courier New, Courier, monospace\">Potentially vulnerable</font></td>" +
		"<td><font face=\"Courier New, Courier, monospace\">&nbsp;" + strconv.Itoa(len(pIssues.potVulns)) + "</font></td>" +
		"</tr>" +
		"<tr>" +
		"<td><font face=\"Courier New, Courier, monospace\">DNS</font></td>" +
		"<td><font face=\"Courier New, Courier, monospace\">&nbsp;" + strconv.Itoa(len(pIssues.DNS)) + "</font></td>" +
		"</tr>" +
		"<tr>" +
		"<td><font face=\"Courier New, Courier, monospace\">Request</font></td>" +
		"<td><font face=\"Courier New, Courier, monospace\">&nbsp;" + strconv.Itoa(len(pIssues.request)) + "</font></td>" +
		"</tr>" +
		"</table>" +
		"<br/><font face=\"Courier New, Courier, monospace\">" +
		"&nbsp;Potentially vulnerable URLs<br/>" +
		"-----------------------------" +
		"<br/>" +
		"</font>" +
		"<table border=\"0\" cellpadding=\"3\" cellspacing=\"4\" width=\"300\">"

	if len(pIssues.potVulns) > 0 {
		for _, vuln := range pIssues.potVulns {
			body += "<tr><td width=\"300\"><font face=\"Courier New, Courier, monospace\">" + vuln.url + "</font></td></tr>"
		}
	} else {
		body += "<tr><td width=\"300\"><font face=\"Courier New, Courier, monospace\">none found</font></td></tr>"
	}
	// close table
	body = body + "</table>"
	msg.SetBody("text/html", body)

	var dnsIssuesFilePath, requestIssuesFilePath string
	if len(pIssues.DNS) > 0 {
		// generate DNS issues file to attach
		dnsIssuesFilePath = generateDNSIssueList(pIssues.DNS)
		msg.Attach(dnsIssuesFilePath)
	}

	if len(pIssues.request) > 0 {
		// generate requests issues file to attach
		requestIssuesFilePath = generateRequestIssueList(pIssues.request)
		msg.Attach(requestIssuesFilePath)
	}

	var emailRaw bytes.Buffer
	_, err = msg.WriteTo(&emailRaw)
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	switch email.Provider {
	case "ses":
		var sess *session.Session
		var staticCreds *credentials.Credentials
		if email.Provider == "ses" {
			if email.AWSAccessKeyID != "" && email.AWSSecretAccessKey != "" && email.AWSSessionToken != "" {
				// try getting with id, secret, and session
				staticCreds = credentials.NewStaticCredentials(email.AWSAccessKeyID,
					email.AWSSecretAccessKey, email.AWSSessionToken)
				sess, err = session.NewSession(&aws.Config{Credentials: staticCreds})
			} else if email.AWSAccessKeyID != "" && email.AWSSecretAccessKey != "" {
				//try with id and secret only
				staticCreds = credentials.NewStaticCredentials(email.AWSAccessKeyID,
					email.AWSSecretAccessKey, "")
				sess, err = session.NewSession(&aws.Config{Credentials: staticCreds})
			} else {
				// try discovering credentials
				sess, err = session.NewSession()
			}
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			err = validateEmailSettings(email)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		msg.SetHeader("To", strings.Join(email.Recipients, ","))
		svc := ses.New(sess, &aws.Config{Region: PtrToStr(email.Region)})
		message := ses.RawMessage{Data: emailRaw.Bytes()}
		source := aws.String(email.Source)
		var destinations []*string
		for _, dest := range email.Recipients {
			destinations = append(destinations, PtrToStr(dest))
		}
		input := ses.SendRawEmailInput{Source: source, Destinations: destinations, RawMessage: &message}
		_, err = svc.SendRawEmail(&input)
		if err != nil {
			panic(err)
		}
	case "smtp":
		msg.SetHeader("To", email.Recipients...)
		host := email.Host
		port, _ := strconv.Atoi(email.Port)
		dialer := gomail.NewPlainDialer(host, port, email.Username, email.Password)
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         host,
		}
		dialer.TLSConfig = tlsConfig
		err = dialer.DialAndSend(msg)
		if err != nil {
			cleanUpFiles(dnsIssuesFilePath, requestIssuesFilePath)
			panic(err)
		}
	}
	return
}

func cleanUpFiles(dnsIssuesFilePath string, requestIssuesFilePath string) {
	if dnsIssuesFilePath != "" {
		delDNSErr := os.Remove(dnsIssuesFilePath)
		if delDNSErr != nil {
			fmt.Println(delDNSErr)
		}
	}
	if requestIssuesFilePath != "" {
		delReqErr := os.Remove(requestIssuesFilePath)
		if delReqErr != nil {
			fmt.Println(delReqErr)
		}
	}
}
