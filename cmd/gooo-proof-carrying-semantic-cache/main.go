package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-proof-carrying-semantic-cache/internal/engine"
	"github.com/kimjooyoon/gooo-proof-carrying-semantic-cache/internal/protocol"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "conformance" {
		fatalf("usage: gooo-proof-carrying-semantic-cache conformance")
	}
	if err := conformance(os.Args[2:]); err != nil {
		fatalf("%v", err)
	}
}

func conformance(args []string) error {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	meta := flags.String("meta", "", "metacode contract")
	corpus := flags.String("corpus", "", "fixed fixture corpus")
	root := flags.String("root", ".", "input repository root")
	out := flags.String("out", "", "caller-owned output directory")
	toolchain := flags.String("toolchain", "", "current Go toolchain identity")
	runner := flags.String("runner-digest", "", "current CI runner identity digest")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *meta == "" || *corpus == "" || *out == "" {
		return fmt.Errorf("meta, corpus, and out are required")
	}
	report, err := engine.RunConformance(engine.Options{
		Root: *root, MetaPath: *meta, CorpusPath: *corpus, OutputPath: *out,
		Toolchain: *toolchain, RunnerDigest: *runner,
	})
	if err != nil {
		return err
	}
	if err := protocol.WriteJSON(*out+"/conformance.json", report); err != nil {
		return err
	}
	if report.Decision != protocol.Closed {
		return fmt.Errorf("fixed corpus did not close: %s", report.Decision)
	}
	fmt.Printf("decision=%s cases=%d closed=%d unknown=%d refuted=%d tests_executed=%d tests_reused=%d\n", report.Decision, report.Summary.TotalCases, report.Summary.Closed, report.Summary.Unknown, report.Summary.Refuted, report.Summary.TestsExecuted, report.Summary.TestsReused)
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
