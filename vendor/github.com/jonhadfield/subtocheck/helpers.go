package subtocheck

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

func padToWidth(input string, trimToWidth bool) (output string) {
	// Split string into lines
	var lines []string
	var newLines []string
	if strings.Contains(input, "\n") {
		lines = strings.Split(input, "\n")
	} else {
		lines = []string{input}
	}
	var paddingSize int
	for i, line := range lines {
		width, _, _ := terminal.GetSize(0)
		if width == -1 {
			width = 80
		}
		// No padding for a line that already meets or exceeds console width
		length := len(line)

		if length >= width {
			if trimToWidth {
				output = line[0:width]
			} else {
				output = input
			}
			return
		} else if i == len(lines)-1 {
			paddingSize = width - len(line)
			if paddingSize >= 1 {
				newLines = append(newLines, fmt.Sprintf("%s%s\r", line, strings.Repeat(" ", paddingSize)))
			} else {
				newLines = append(newLines, fmt.Sprintf("%s\r", line))
			}
		} else {
			var suffix string
			newLines = append(newLines, fmt.Sprintf("%s%s%s\n", line, strings.Repeat(" ", paddingSize), suffix))
		}
	}
	output = strings.Join(newLines, "")
	return
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetStringInBetween(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str, end)
	return str[s:e]
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func StringSliceToLower(input []string) (output []string) {
	output = Map(input, strings.ToLower)
	return
}

func StringInSliceContents(a string, list []string) bool {
	for _, b := range list {
		if strings.Contains(a, b) {
			return true
		}
	}
	return false
}

func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

// PtrToStr returns a pointer to an existing string
func PtrToStr(s string) *string {
	return &s
}
