package main

import (
	"fmt"
	"io"

	"github.com/easel/dun/internal/dun"
)

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "usage: dun doctor")
		return dun.ExitUsageError
	}
	root := resolveRoot(".")
	report, err := dun.RunDoctor(root)
	fmt.Fprint(stdout, dun.FormatDoctorReport(report))
	if err != nil {
		fmt.Fprintf(stderr, "dun doctor failed: %v\n", err)
		return dun.ExitRuntimeError
	}
	return dun.ExitSuccess
}
