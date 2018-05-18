package subtocheck

import "fmt"

type processedIssues struct {
	potVulns []issue
	DNS      []issue
	request  []issue
}

func getIssuesSummary(issues issues) (pIssues processedIssues) {
	for _, issue := range issues {
		switch issue.kind {
		case "request":
			pIssues.request = append(pIssues.request, issue)
		case "dns":
			pIssues.DNS = append(pIssues.DNS, issue)
		case "vuln":
			pIssues.potVulns = append(pIssues.potVulns, issue)
		}
	}
	return
}

func displayIssues(pIssues processedIssues) {
	var txtNoIssuesFound = "none found"

	fmt.Printf("\nRequest issues\n--------------\n")
	if len(pIssues.request) > 0 {
		for _, issue := range pIssues.request {
			if issue.kind == "request" {
				fmt.Printf("%v\n", issue.err)
			}
		}
	} else {
		fmt.Println(txtNoIssuesFound)
	}

	fmt.Printf("\nDNS issues\n----------\n")
	if len(pIssues.DNS) > 0 {
		for _, issue := range pIssues.DNS {
			if issue.kind == "dns" {
				fmt.Printf("%v\n", issue.err)
			}
		}
	} else {
		fmt.Println(txtNoIssuesFound)
	}
	fmt.Printf("\nPotential vulnerabilities\n-------------------------\n")
	if len(pIssues.potVulns) > 0 {
		for _, issue := range pIssues.potVulns {
			if issue.kind == "vuln" {
				fmt.Printf("%s %v\n", issue.url, issue.err)
			}
		}
	} else {
		fmt.Println(txtNoIssuesFound)
	}
}
