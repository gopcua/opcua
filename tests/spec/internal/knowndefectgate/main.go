package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2/types"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: knowndefectgate <ginkgo-json-report> <go-test-log>")
		return 1
	}
	reportData, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var reports []types.Report
	if err := json.Unmarshal(reportData, &reports); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	logData, err := os.ReadFile(args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "knowndefectgate: %v\n", err)
		fmt.Println("the log was not read, so the data-race and timeout-panic rules did not run")
		return 1
	}
	exit, message := Verdict(reports, string(logData))
	if message != "" {
		fmt.Println(message)
	}
	return exit
}
