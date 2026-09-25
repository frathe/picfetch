// Command locationmapqualify collects and validates native Location Map evidence.
package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"
)

func main() {
	if err := runCLI(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func binaryIdentity(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", digest.Sum(nil)), nil
}

func runCLI(args []string, output io.Writer) error {
	if len(args) > 0 && args[0] == "run" {
		options, err := parseNativeRun(args[1:], output)
		if err != nil {
			return err
		}
		interrupt, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		ctx, cancel := context.WithTimeout(interrupt, options.timeout)
		defer cancel()
		return runNative(ctx, options.images, options.evidence, options.binary, options.helper, output)
	}
	if len(args) == 0 || args[0] != "check" {
		return errors.New("usage: locationmapqualify check -evidence DIR -images COUNT -binary FILE; or run -images DIR -evidence NEW_DIR -binary FILE -helper FILE [-timeout 30m]")
	}
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(output)
	dir := flags.String("evidence", "", "native evidence directory")
	images := flags.Int("images", 0, "required admitted image count")
	binary := flags.String("binary", "", "current application binary to verify")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *dir == "" || *images <= 0 || *binary == "" {
		return errors.New("evidence, positive image count and current binary are required")
	}
	build, err := binaryIdentity(*binary)
	if err != nil {
		return fmt.Errorf("build identity: %w", err)
	}
	report, err := CheckEvidence(*dir, *images, build)
	if err != nil {
		return err
	}
	var peak int64
	for _, sample := range report.Memory {
		peak = max(peak, sample.RSSBytes)
	}
	_, err = fmt.Fprintf(output, "Valid native evidence: %d images, %d gestures, peak RSS %d bytes, build %s\n", report.Images, len(report.Gestures), peak, build)
	return err
}

type nativeRunOptions struct {
	images, evidence, binary, helper string
	timeout                          time.Duration
}

func parseNativeRun(args []string, output io.Writer) (nativeRunOptions, error) {
	var options nativeRunOptions
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&options.images, "images", "", "explicit image collection directory")
	flags.StringVar(&options.evidence, "evidence", "", "new evidence directory")
	flags.StringVar(&options.binary, "binary", "", "native application binary")
	flags.StringVar(&options.helper, "helper", "", "compiled native screen/input observer")
	flags.DurationVar(&options.timeout, "timeout", 30*time.Minute, "finite whole-run deadline")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() != 0 || options.images == "" || options.evidence == "" || options.binary == "" || options.helper == "" || options.timeout <= 0 {
		return options, errors.New("explicit image/evidence directories, binary, helper and positive timeout are required")
	}
	return options, nil
}
