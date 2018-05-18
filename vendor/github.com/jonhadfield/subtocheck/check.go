package subtocheck

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"reflect"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	httpPrefix   = "http://"
	httpsPrefix  = "https://"
	protocols    = []string{"http", "https"}
	resolveMutex sync.Mutex
	nameservers  = []string{
		"8.8.8.8",         // google
		"8.8.4.4",         // google
		"209.244.0.3",     // level3
		"209.244.0.4",     // level3
		"1.1.1.1",         // cloudflare
		"1.0.0.1",         // cloudflare
		"9.9.9.9",         // quad9
		"149.112.112.112", // quad9
	}
)

type issue struct {
	kind string // vuln, request, dns
	fqdn string
	url  string
	err  error
}

type issues []issue

func checkResolves(fqdn string, debug *bool) (issues issues) {
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	m.RecursionDesired = true
	c.Timeout = 1500 * time.Millisecond
	var record *dns.Msg
	var err error
	resolveMutex.Lock()
	rand.Seed(time.Now().UnixNano())
	ns := rand.Int() % len(nameservers)
	if *debug {
		fmt.Printf("DEBUG: resolving \"%s\" with nameserver %s\n", fqdn, nameservers[ns])
	}
	record, _, err = c.Exchange(m, net.JoinHostPort(nameservers[ns], strconv.Itoa(53)))
	resolveMutex.Unlock()
	if err != nil {
		err = errors.Errorf("%s could not be resolved (%v)", fqdn, err)
		issues = append(issues, issue{kind: "dns", fqdn: fqdn, err: err})
	} else if len(record.Answer) == 0 {
		err = errors.Errorf("%s could not be resolved (no answer from %s)", fqdn, nameservers[ns])
		issues = append(issues, issue{kind: "dns", fqdn: fqdn, err: err})
	} else if record.Rcode != 0 {
		err = errors.Errorf("%s could not be resolved (%s from %s)", fqdn, dns.RcodeToString[record.Rcode],
			nameservers[ns])
		issues = append(issues, issue{kind: "dns", fqdn: fqdn, err: err})
	}
	if *debug {
		fmt.Printf("DEBUG: error: %v\n", err)
	}

	return
}

func checkResponse(fqdn string, protocols []string, debug *bool) (issues issues) {
	var clientTransportTimeoutSecs = 3
	var responseHeaderTimeoutSecs = 2

	tr := &http.Transport{
		ResponseHeaderTimeout: time.Duration(responseHeaderTimeoutSecs) * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(clientTransportTimeoutSecs) * time.Second,
	}
	for _, protocol := range protocols {
		var httpURL string
		if protocol == "http" {
			httpURL = httpPrefix + fqdn
		} else if protocol == "https" {
			httpURL = httpsPrefix + fqdn
		}
		var httpResp *http.Response
		var err error
		if *debug {
			fmt.Printf("DEBUG: requesting URL \"%s\" with client transport timeout: %d secs and resp. header"+
				" timeout: %d secs\n", httpURL, clientTransportTimeoutSecs, responseHeaderTimeoutSecs)
		}
		httpResp, err = client.Get(httpURL)
		if err != nil {
			issues = append(issues, issue{kind: "request", fqdn: fqdn, err: err})
		}
		if httpResp != nil {
			vulnIssue := checkVulnerable(httpURL, httpResp)
			if vulnIssue.kind != "" {
				issues = append(issues, vulnIssue)
			}
		}
	}
	return
}

type vPattern struct {
	platform        string
	responseCodes   []int // 0 for all
	bodyStrings     []string
	bodyStringMatch string
}

var vPatterns = []vPattern{
	{
		platform:        "CloudFront",
		responseCodes:   []int{403},
		bodyStrings:     []string{"The request could not be satisfied."},
		bodyStringMatch: "all",
	},
	{
		platform:        "Heroku",
		responseCodes:   []int{404},
		bodyStrings:     []string{"//www.herokucdn.com/error-pages/no-such-app.html"},
		bodyStringMatch: "all",
	},
	{
		platform:        "S3",
		responseCodes:   []int{404},
		bodyStrings:     []string{"Code: NoSuchBucket"},
		bodyStringMatch: "all",
	},
	{
		platform:        "Tumblr",
		responseCodes:   []int{404},
		bodyStrings:     []string{"Not found.", "assets.tumblr.com"},
		bodyStringMatch: "all",
	},
}

func checkVulnerable(url string, response *http.Response) (vuln issue) {
	for _, pattern := range vPatterns {
		if len(pattern.responseCodes) > 0 {
			if pattern.responseCodes == nil || !contains(pattern.responseCodes, response.StatusCode) {
				continue
			}
		}
		if checkBodyResponse(pattern, response.Body) {
			return issue{
				url:  url,
				kind: "vuln",
				err:  errors.Errorf("matches pattern for platform: %s", pattern.platform),
			}
		}
	}
	return
}

func checkBodyResponse(pattern vPattern, body io.ReadCloser) (result bool) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(body)
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	bodyText := buf.String()
	for _, bodyString := range pattern.bodyStrings {
		if strings.Contains(bodyText, bodyString) {
			result = true
		} else if pattern.bodyStringMatch == "all" {
			result = false
			return
		}
	}
	return
}

var domainIssues issues

func CheckDomains(path string, configPath *string, debug *bool, quiet *bool) {
	var conf Config
	if *configPath != "" {
		conf = readConfig(*configPath)
	}
	file, _ := os.Open(path)
	domainScanner := bufio.NewScanner(file)
	var domains []string
	for domainScanner.Scan() {
		entry := domainScanner.Text()
		if entry != "" {
			domains = append(domains, entry)
		}
	}
	jobs := make(chan string, len(domains))
	results := make(chan bool, len(domains))

	for w := 1; w <= 10; w++ {
		go worker(w, jobs, results, debug)
	}
	numDomains := len(domains)
	for j := 0; j < numDomains; j++ {
		jobs <- domains[j]
	}
	close(jobs)

	var progress string
	for a := 1; a <= numDomains; a++ {
		if !*quiet {
			progress = fmt.Sprintf("Processing... %d/%d %s", a, numDomains, domains[a-1])
			progress = padToWidth(progress, true)
			width, _, _ := terminal.GetSize(0)
			if len(progress) == width {
				fmt.Printf(progress[0:width-3] + "   \r")
			} else {
				fmt.Print(progress)
			}
		}

		<-results
	}
	pIssues := getIssuesSummary(domainIssues)
	if !*quiet {
		fmt.Printf("%s", padToWidth(" ", false))
		if !reflect.DeepEqual(pIssues, processedIssues{}) {
			displayIssues(pIssues)
		} else {
			fmt.Println("\nno issues found.")
		}
	}
	// send notifications
	if conf.Email.Provider != "" {
		if *debug {
			fmt.Println("\nDEBUG: sending email")
		}
		emailErr := emailResults(conf.Email, pIssues)
		if emailErr != nil {
			fmt.Println("failed to send email")
			fmt.Println("-- error --")
			fmt.Printf("%+v\n", emailErr)
		}
	}
}

func worker(id int, jobs <-chan string, results chan<- bool, debug *bool) {
	for j := range jobs {
		if *debug {
			fmt.Printf("DEBUG: worker: %d\n", id)
		}
		resolveIssues := checkResolves(j, debug)
		if len(resolveIssues) > 0 {
			domainIssues = append(domainIssues, resolveIssues...)
		} else {
			responseIssues := checkResponse(j, protocols, debug)
			if len(responseIssues) > 0 {
				domainIssues = append(domainIssues, responseIssues...)
			}
		}
		results <- true
	}
}
