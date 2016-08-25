package terminator

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

// Shortcut
type args map[string]interface{}

// E.g.: logger("INFO", args{"uuid":*node.UUID, "error":err})
// Outputs: 'level'='INFO' 'uuid'='caa1dd48-bd8a-4bc0-907a-76fa0207ce33' 'error'='Not found'
func logger(level string, params args) {
	var logs []string
	for k, v := range params {
		logs = append(logs, fmt.Sprintf("%q=%q", k, v))
	}
	// Make the output consistent
	sort.Strings(logs)
	// Make sure that the line number is set from the calling stack frame
	log.Output(2, strings.ToUpper(level)+" "+strings.Join(logs, " "))
	if level == "FATAL" || level == "fatal" {
		os.Exit(1)
	}
}

// Log prints the args as a single value to a "message" key
func Log(level string, inputs ...interface{}) {
	logger(level, args{"message": fmt.Sprintln(inputs...)})
}
