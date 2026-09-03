// Command testshards summarizes and captures Go test events, plans deterministic
// shards, validates live assignments, partitions packages, and emits exact shard
// filters.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

type pathsFlag []string

func (p *pathsFlag) String() string {
	return strings.Join(*p, ",")
}

func (p *pathsFlag) Set(value string) error {
	if value == "" {
		return errors.New("path must not be empty")
	}
	*p = append(*p, value)
	return nil
}

type goEvent struct {
	Action  string   `json:"Action"`
	Package string   `json:"Package"`
	Test    string   `json:"Test"`
	Output  string   `json:"Output"`
	Elapsed *float64 `json:"Elapsed"`
}

type terminal struct {
	action  string
	elapsed time.Duration
}

type testKey struct {
	packageName string
	name        string
}

type streamSummary struct {
	packages map[string]terminal
	tests    map[testKey]terminal
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "testshards: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	return runWithInput(args, os.Stdin, stdout, stderr)
}

func runWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: testshards <summarize|plan|check|regex|capture|partition> [arguments]")
	}

	switch args[0] {
	case "summarize":
		return runSummarize(args[1:], stdout, stderr)
	case "plan":
		return runPlan(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "regex":
		return runRegex(args[1:], stdout, stderr)
	case "capture":
		return runCapture(args[1:], stdin, stdout, stderr)
	case "partition":
		return runPartition(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runPartition(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("partition", flag.ContinueOnError)
	fs.SetOutput(stderr)
	packageArg := fs.String("package", "", "exact package to remove from the module inventory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *packageArg == "" {
		return errors.New("usage: testshards partition -package PACKAGE")
	}

	packages, err := loadPackagePartition(*packageArg)
	if err != nil {
		return err
	}
	for _, packageName := range packages {
		if _, err := fmt.Fprintln(stdout, packageName); err != nil {
			return err
		}
	}
	return nil
}

func loadPackagePartition(packageArg string) ([]string, error) {
	if strings.HasPrefix(packageArg, "-") || strings.Contains(packageArg, "...") {
		return nil, fmt.Errorf("package %s must name one exact package", packageArg)
	}
	selectedOutput, err := goCommandOutput("list", "-f", "{{.ImportPath}}", "--", packageArg)
	if err != nil {
		return nil, err
	}
	selected, err := packageNames(selectedOutput, "selected package")
	if err != nil {
		return nil, err
	}
	if len(selected) != 1 {
		return nil, fmt.Errorf("package %s resolved to %d packages; want exactly one", packageArg, len(selected))
	}

	moduleOutput, err := goCommandOutput("list", "-f", "{{.ImportPath}}", "--", "./...")
	if err != nil {
		return nil, err
	}
	modulePackages, err := packageNames(moduleOutput, "module inventory")
	if err != nil {
		return nil, err
	}
	return partitionPackageNames(modulePackages, selected[0])
}

func packageNames(output []byte, source string) ([]string, error) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, fmt.Errorf("%s is empty", source)
	}
	for index := range lines {
		if lines[index] == "" || strings.TrimSpace(lines[index]) != lines[index] || strings.ContainsAny(lines[index], "\t\r ") {
			return nil, fmt.Errorf("%s contains malformed package name %q", source, lines[index])
		}
	}
	return lines, nil
}

func partitionPackageNames(modulePackages []string, selected string) ([]string, error) {
	seen := make(map[string]struct{}, len(modulePackages))
	partition := make([]string, 0, len(modulePackages))
	selectedFound := false
	for _, packageName := range modulePackages {
		if _, exists := seen[packageName]; exists {
			return nil, fmt.Errorf("module inventory contains duplicate package %s", packageName)
		}
		seen[packageName] = struct{}{}
		if packageName == selected {
			selectedFound = true
			continue
		}
		partition = append(partition, packageName)
	}
	if !selectedFound {
		return nil, fmt.Errorf("selected package %s is absent from the module inventory", selected)
	}
	if len(partition) == 0 {
		return nil, errors.New("non-UI package partition is empty")
	}
	sort.Strings(partition)
	return partition, nil
}

func runCapture(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("out", "", "path that receives the raw go test -json stream")
	partition := fs.String("partition", "", "partition label included in concise output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *outPath == "" || *partition == "" {
		return errors.New("usage: testshards capture -out PATH -partition NAME")
	}
	if strings.TrimSpace(*partition) != *partition || strings.ContainsAny(*partition, "\t\r\n") {
		return fmt.Errorf("partition %q must be a non-empty single field", *partition)
	}

	raw, err := os.Create(*outPath)
	if err != nil {
		return fmt.Errorf("create capture %s: %w", *outPath, err)
	}
	captureErr := captureStream(stdin, raw, stdout, *partition)
	if err := raw.Close(); err != nil {
		captureErr = errors.Join(captureErr, fmt.Errorf("close capture %s: %w", *outPath, err))
	}
	return captureErr
}

func captureStream(input io.Reader, raw io.Writer, concise io.Writer, partition string) error {
	reader := bufio.NewReader(input)
	pendingOutput := make(map[testKey][]string)
	eventCount := 0
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) != 0 {
			if _, err := raw.Write(line); err != nil {
				return fmt.Errorf("write raw event stream: %w", err)
			}
			eventCount++
			var event goEvent
			if err := json.Unmarshal(bytes.TrimSpace(line), &event); err != nil {
				return finishCapture(reader, raw, fmt.Errorf("parse captured event at line %d: %w", eventCount, err))
			}
			if event.Action == "output" && event.Output != "" {
				key := testKey{packageName: event.Package, name: event.Test}
				pendingOutput[key] = append(pendingOutput[key], captureOutputLines(event.Output)...)
			}
			if isTerminal(event.Action) {
				key := testKey{packageName: event.Package, name: event.Test}
				if event.Action == "fail" {
					for _, output := range pendingOutput[key] {
						if err := writeCaptureFailure(concise, partition, key, output); err != nil {
							return finishCapture(reader, raw, err)
						}
					}
				}
				delete(pendingOutput, key)
				if event.Test == "" {
					if err := writeCaptureTerminal(concise, "package", partition, event.Package, "", event.Action, event.Elapsed); err != nil {
						return finishCapture(reader, raw, err)
					}
				} else if event.Action == "fail" || isTopLevelRunnable(event.Test) {
					if err := writeCaptureTerminal(concise, "test", partition, event.Package, event.Test, event.Action, event.Elapsed); err != nil {
						return finishCapture(reader, raw, err)
					}
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return fmt.Errorf("read event stream: %w", readErr)
		}
	}
	if eventCount == 0 {
		return errors.New("captured stream contains no Go test events")
	}
	return nil
}

func finishCapture(input io.Reader, raw io.Writer, captureErr error) error {
	if _, err := io.Copy(raw, input); err != nil {
		return errors.Join(captureErr, fmt.Errorf("preserve remaining raw event stream: %w", err))
	}
	return captureErr
}

func captureOutputLines(output string) []string {
	output = strings.TrimRight(output, "\r\n")
	if output == "" {
		return nil
	}
	lines := strings.Split(output, "\n")
	for index := range lines {
		lines[index] = strings.TrimSuffix(lines[index], "\r")
	}
	return lines
}

func writeCaptureFailure(out io.Writer, partition string, key testKey, output string) error {
	if key.name == "" {
		_, err := fmt.Fprintf(out, "failure\tpartition=%s\tpackage=%s\t%s\n", partition, key.packageName, output)
		return err
	}
	_, err := fmt.Fprintf(out, "failure\tpartition=%s\tpackage=%s\ttest=%s\t%s\n", partition, key.packageName, key.name, output)
	return err
}

func writeCaptureTerminal(out io.Writer, kind, partition, packageName, testName, action string, elapsed *float64) error {
	elapsedText := "unknown"
	if elapsed != nil {
		elapsedText = strconv.FormatFloat(*elapsed, 'f', 3, 64)
	}
	if testName == "" {
		_, err := fmt.Fprintf(out, "%s\tpartition=%s\tpackage=%s\taction=%s\telapsed=%s\n", kind, partition, packageName, action, elapsedText)
		return err
	}
	_, err := fmt.Fprintf(out, "%s\tpartition=%s\tpackage=%s\tname=%s\taction=%s\telapsed=%s\n", kind, partition, packageName, testName, action, elapsedText)
	return err
}

func runSummarize(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("summarize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var paths pathsFlag
	fs.Var(&paths, "json", "path to go test -json output (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || len(paths) == 0 {
		return errors.New("usage: testshards summarize -json PATH [-json PATH ...]")
	}

	summaries, err := loadComparableStreams(paths)
	if err != nil {
		return err
	}
	return writeSummary(stdout, summaries)
}

func runPlan(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var paths pathsFlag
	fs.Var(&paths, "json", "path to go test -json output (must be repeated three times)")
	packageName := fs.String("package", "", "exact package import path represented by the manifest")
	shardCount := fs.Int("shards", 0, "number of shards")
	baselineSHA := fs.String("baseline-sha", "", "source commit shared by the baseline attempts")
	baselineRun := fs.String("baseline-run", "", "GitHub Actions baseline run ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || len(paths) != 3 || *packageName == "" || *shardCount < 1 || *baselineSHA == "" || *baselineRun == "" {
		return errors.New("usage: testshards plan -package PACKAGE -shards N -baseline-sha SHA -baseline-run RUN -json RUN1 -json RUN2 -json RUN3")
	}

	summaries, err := loadComparableStreams(paths)
	if err != nil {
		return err
	}
	return writePlan(stdout, summaries, *packageName, *shardCount, *baselineSHA, *baselineRun)
}

func runRegex(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("regex", flag.ContinueOnError)
	fs.SetOutput(stderr)
	manifestPath := fs.String("manifest", "", "path to the shard manifest")
	shardName := fs.String("shard", "", "shard whose exact-match filter to emit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *manifestPath == "" || *shardName == "" {
		return errors.New("usage: testshards regex -manifest PATH -shard ui-N")
	}

	manifest, err := readShardManifest(*manifestPath)
	if err != nil {
		return err
	}
	names, exists := manifest.namesByShard[*shardName]
	if !exists {
		return fmt.Errorf("manifest %s has no shard %s", *manifestPath, *shardName)
	}

	sort.Strings(names)
	quoted := make([]string, len(names))
	for index, name := range names {
		quoted[index] = regexp.QuoteMeta(name)
	}
	filter := "^(" + strings.Join(quoted, "|") + ")$"
	if filter == "^()$" {
		return fmt.Errorf("manifest shard %s is empty", *shardName)
	}
	if _, err := regexp.Compile(filter); err != nil {
		return fmt.Errorf("compile filter for shard %s: %w", *shardName, err)
	}
	_, err = fmt.Fprintln(stdout, filter)
	return err
}

func runCheck(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	packageArg := fs.String("package", "", "exact package whose runnable inventory to validate")
	manifestPath := fs.String("manifest", "", "path to the shard manifest")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *packageArg == "" || *manifestPath == "" {
		return errors.New("usage: testshards check -package PACKAGE -manifest PATH")
	}

	manifest, err := readShardManifest(*manifestPath)
	if err != nil {
		return err
	}
	buildContext, err := currentGoBuildContext()
	if err != nil {
		return err
	}
	if buildContext.GOOS != runtime.GOOS || buildContext.GOARCH != runtime.GOARCH {
		return fmt.Errorf("go build context %s/%s does not match running testshards executable %s/%s", buildContext.GOOS, buildContext.GOARCH, runtime.GOOS, runtime.GOARCH)
	}
	inventory, err := loadRunnableInventory(*packageArg)
	if err != nil {
		return err
	}
	if manifest.packageName != inventory.packageName {
		return fmt.Errorf("manifest package %s does not match selected package %s", manifest.packageName, inventory.packageName)
	}
	if err := validateExactAssignment(manifest, inventory.names); err != nil {
		return err
	}
	if err := rejectParallelCalls(inventory); err != nil {
		return err
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("canonical shard validation requires linux/amd64; running testshards executable is %s/%s (use make check-test-shards)", runtime.GOOS, runtime.GOARCH)
	}

	_, err = fmt.Fprintf(stdout, "checked\tpackage=%s\trunnables=%d\tshards=%d\n", inventory.packageName, len(inventory.names), len(manifest.namesByShard))
	return err
}

type goBuildContext struct {
	GOOS   string
	GOARCH string
}

func currentGoBuildContext() (goBuildContext, error) {
	output, err := goCommandOutput("env", "-json", "GOOS", "GOARCH")
	if err != nil {
		return goBuildContext{}, err
	}
	var context goBuildContext
	if err := json.Unmarshal(output, &context); err != nil {
		return goBuildContext{}, fmt.Errorf("parse go env build context: %w", err)
	}
	if context.GOOS == "" || context.GOARCH == "" {
		return goBuildContext{}, errors.New("go env returned an empty GOOS or GOARCH")
	}
	return context, nil
}

type listedPackage struct {
	Dir          string
	ImportPath   string
	TestGoFiles  []string
	XTestGoFiles []string
}

type runnableInventory struct {
	packageName string
	dir         string
	testFiles   []string
	names       map[string]struct{}
}

func loadRunnableInventory(packageArg string) (runnableInventory, error) {
	if strings.HasPrefix(packageArg, "-") || strings.Contains(packageArg, "...") {
		return runnableInventory{}, fmt.Errorf("package %s must name one exact package", packageArg)
	}
	output, err := goCommandOutput("list", "-json", "--", packageArg)
	if err != nil {
		return runnableInventory{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	var listed listedPackage
	if err := decoder.Decode(&listed); err != nil {
		return runnableInventory{}, fmt.Errorf("parse go list result for %s: %w", packageArg, err)
	}
	var extra listedPackage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return runnableInventory{}, fmt.Errorf("package %s resolved to more than one package", packageArg)
		}
		return runnableInventory{}, fmt.Errorf("parse trailing go list result for %s: %w", packageArg, err)
	}
	if listed.Dir == "" || listed.ImportPath == "" {
		return runnableInventory{}, fmt.Errorf("go list returned incomplete package metadata for %s", packageArg)
	}
	testFiles := append([]string(nil), listed.TestGoFiles...)
	testFiles = append(testFiles, listed.XTestGoFiles...)
	sort.Strings(testFiles)
	if len(testFiles) == 0 {
		return runnableInventory{}, fmt.Errorf("package %s has no build-selected test files", listed.ImportPath)
	}

	output, err = goCommandOutput("test", "-run", "^$", "-list", "^(Test|Fuzz|Example)", packageArg)
	if err != nil {
		return runnableInventory{}, err
	}
	names := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "TestMain" || !isTopLevelRunnable(name) {
			continue
		}
		if _, exists := names[name]; exists {
			return runnableInventory{}, fmt.Errorf("go test -list returned duplicate runnable %s", name)
		}
		names[name] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return runnableInventory{}, fmt.Errorf("read go test -list output for %s: %w", listed.ImportPath, err)
	}
	if len(names) == 0 {
		return runnableInventory{}, fmt.Errorf("package %s has no top-level tests, fuzz targets, or examples", listed.ImportPath)
	}
	return runnableInventory{
		packageName: listed.ImportPath,
		dir:         listed.Dir,
		testFiles:   testFiles,
		names:       names,
	}, nil
}

func validateExactAssignment(manifest shardManifest, inventory map[string]struct{}) error {
	names := make([]string, 0, len(inventory))
	for name := range inventory {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, assigned := manifest.assignment[name]; !assigned {
			return fmt.Errorf("current runnable %s is unassigned in the shard manifest", name)
		}
	}

	names = names[:0]
	for name := range manifest.assignment {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, current := inventory[name]; !current {
			return fmt.Errorf("manifest test %s is stale: absent from the current runnable inventory", name)
		}
	}
	return nil
}

func rejectParallelCalls(inventory runnableInventory) error {
	files := append([]string(nil), inventory.testFiles...)
	sort.Strings(files)
	for _, name := range files {
		if filepath.Base(name) != name {
			return fmt.Errorf("go list returned invalid selected test file %q", name)
		}
		fileSet := token.NewFileSet()
		filePath := filepath.Join(inventory.dir, name)
		syntax, err := parser.ParseFile(fileSet, filePath, nil, 0)
		if err != nil {
			return fmt.Errorf("parse selected test file %s: %w", filePath, err)
		}
		for _, declaration := range syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			var parallel *ast.CallExpr
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if parallel != nil {
					return false
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if ok && selector.Sel.Name == "Parallel" {
					parallel = call
					return false
				}
				return true
			})
			if parallel != nil {
				position := fileSet.Position(parallel.Pos())
				return fmt.Errorf("selected test file %s:%d calls .Parallel() in %s; explicit safety review is required before exact UI package tests run in parallel", name, position.Line, function.Name.Name)
			}
		}
	}
	return nil
}

func goCommandOutput(args ...string) ([]byte, error) {
	command := exec.Command("go", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err == nil {
		return output, nil
	}
	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		return nil, fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
	}
	return nil, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, detail)
}

func loadComparableStreams(paths []string) ([]streamSummary, error) {
	summaries := make([]streamSummary, 0, len(paths))
	for _, singlePath := range paths {
		summary, err := readEventStream(singlePath)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	if err := validateComparable(summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

func validateComparable(summaries []streamSummary) error {
	packageRuns := make(map[string]int)
	testRuns := make(map[testKey]int)
	for runIndex, summary := range summaries {
		for name := range summary.packages {
			if _, exists := packageRuns[name]; !exists {
				packageRuns[name] = runIndex
			}
		}
		for name := range summary.tests {
			if _, exists := testRuns[name]; !exists {
				testRuns[name] = runIndex
			}
		}
	}

	packageNames := make([]string, 0, len(packageRuns))
	for name := range packageRuns {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)
	for _, name := range packageNames {
		for runIndex, summary := range summaries {
			if _, exists := summary.packages[name]; !exists {
				return fmt.Errorf("run %d: missing package %s observed in run %d", runIndex+1, name, packageRuns[name]+1)
			}
		}
	}

	testNames := make([]testKey, 0, len(testRuns))
	for name := range testRuns {
		testNames = append(testNames, name)
	}
	sortTestKeys(testNames)
	for _, name := range testNames {
		for runIndex, summary := range summaries {
			if _, exists := summary.tests[name]; !exists {
				return fmt.Errorf("run %d: missing top-level test %s/%s observed in run %d", runIndex+1, name.packageName, name.name, testRuns[name]+1)
			}
		}
	}
	return nil
}

func readEventStream(path string) (streamSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return streamSummary{}, fmt.Errorf("read %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	summary := streamSummary{
		packages: make(map[string]terminal),
		tests:    make(map[testKey]terminal),
	}
	packageStarts := make(map[string]struct{})
	testRuns := make(map[testKey]struct{})
	testTerminals := make(map[testKey]struct{})
	decoder := json.NewDecoder(file)
	eventCount := 0
	for {
		var event goEvent
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return streamSummary{}, fmt.Errorf("parse %s at byte %d: %w", path, decoder.InputOffset(), err)
		}
		eventCount++
		if event.Action == "start" && event.Test == "" {
			packageStarts[event.Package] = struct{}{}
			continue
		}
		if event.Action == "run" && event.Test != "" {
			testRuns[testKey{packageName: event.Package, name: event.Test}] = struct{}{}
			continue
		}
		if !isTerminal(event.Action) {
			continue
		}
		if event.Elapsed == nil {
			if event.Test == "" {
				return streamSummary{}, fmt.Errorf("%s: terminal %s for package %s is missing Elapsed", path, event.Action, event.Package)
			}
			return streamSummary{}, fmt.Errorf("%s: terminal %s for %s/%s is missing Elapsed", path, event.Action, event.Package, event.Test)
		}
		elapsed, err := elapsedDuration(*event.Elapsed)
		if err != nil {
			return streamSummary{}, fmt.Errorf("%s: invalid Elapsed for %s/%s: %w", path, event.Package, event.Test, err)
		}
		result := terminal{action: event.Action, elapsed: elapsed}
		if event.Test == "" {
			if _, exists := packageStarts[event.Package]; !exists {
				return streamSummary{}, fmt.Errorf("%s: terminal outcome for package %s has no start event", path, event.Package)
			}
			if _, exists := summary.packages[event.Package]; exists {
				return streamSummary{}, fmt.Errorf("%s: duplicate terminal outcome for package %s", path, event.Package)
			}
			summary.packages[event.Package] = result
			continue
		}
		key := testKey{packageName: event.Package, name: event.Test}
		if _, exists := testRuns[key]; !exists {
			return streamSummary{}, fmt.Errorf("%s: terminal outcome for %s/%s has no run event", path, event.Package, event.Test)
		}
		if _, exists := testTerminals[key]; exists {
			return streamSummary{}, fmt.Errorf("%s: duplicate terminal outcome for %s/%s", path, event.Package, event.Test)
		}
		testTerminals[key] = struct{}{}
		if isTopLevelRunnable(event.Test) {
			summary.tests[key] = result
		}
	}
	if eventCount == 0 {
		return streamSummary{}, fmt.Errorf("%s: contains no Go test events", path)
	}
	if len(packageStarts) == 0 {
		return streamSummary{}, fmt.Errorf("%s: contains no package start events", path)
	}

	packageNames := make([]string, 0, len(packageStarts))
	for name := range packageStarts {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)
	for _, name := range packageNames {
		if _, exists := summary.packages[name]; !exists {
			return streamSummary{}, fmt.Errorf("%s: missing terminal outcome for package %s", path, name)
		}
	}

	testNames := make([]testKey, 0, len(testRuns))
	for name := range testRuns {
		testNames = append(testNames, name)
	}
	sortTestKeys(testNames)
	for _, name := range testNames {
		if _, exists := testTerminals[name]; !exists {
			return streamSummary{}, fmt.Errorf("%s: missing terminal outcome for %s/%s", path, name.packageName, name.name)
		}
	}
	return summary, nil
}

func isTerminal(action string) bool {
	return action == "pass" || action == "fail" || action == "skip"
}

func isTopLevelRunnable(name string) bool {
	if strings.Contains(name, "/") {
		return false
	}
	return strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Fuzz") || strings.HasPrefix(name, "Example")
}

func writeSummary(out io.Writer, summaries []streamSummary) error {
	if _, err := fmt.Fprintf(out, "summary\truns=%d\n", len(summaries)); err != nil {
		return err
	}

	packageNames := make([]string, 0, len(summaries[0].packages))
	for name := range summaries[0].packages {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)
	for _, name := range packageNames {
		values := make([]terminal, len(summaries))
		for index, summary := range summaries {
			values[index] = summary.packages[name]
		}
		if err := writeResultRow(out, []string{"package", name}, values); err != nil {
			return err
		}
	}

	testNames := make([]testKey, 0, len(summaries[0].tests))
	for name := range summaries[0].tests {
		testNames = append(testNames, name)
	}
	sortTestKeys(testNames)
	for _, name := range testNames {
		values := make([]terminal, len(summaries))
		for index, summary := range summaries {
			values[index] = summary.tests[name]
		}
		if err := writeResultRow(out, []string{"test", name.packageName, name.name}, values); err != nil {
			return err
		}
	}
	return nil
}

type weightedTest struct {
	name   string
	weight time.Duration
}

type shardPlan struct {
	name  string
	load  time.Duration
	tests []string
}

const (
	manifestMagic       = "# PicFetch test shard manifest v1"
	expectedShardCount  = 3
	manifestPackageLine = "# package: "
	manifestShardsLine  = "# shards: "
)

type shardManifest struct {
	packageName   string
	assignment    map[string]string
	assignmentRow map[string]int
	namesByShard  map[string][]string
}

func readShardManifest(manifestPath string) (shardManifest, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return shardManifest{}, fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}
	defer func() { _ = file.Close() }()

	manifest := shardManifest{
		assignment:    make(map[string]string),
		assignmentRow: make(map[string]int),
		namesByShard:  make(map[string][]string),
	}
	packageRow := 0
	shardsRow := 0
	shardCount := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	row := 0
	for scanner.Scan() {
		row++
		line := scanner.Text()
		if row == 1 {
			if line != manifestMagic {
				return shardManifest{}, fmt.Errorf("manifest %s row 1 is malformed: want %q", manifestPath, manifestMagic)
			}
			continue
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			switch {
			case strings.HasPrefix(line, manifestPackageLine):
				if packageRow != 0 {
					return shardManifest{}, fmt.Errorf("manifest %s row %d duplicates package header from row %d", manifestPath, row, packageRow)
				}
				manifest.packageName = strings.TrimPrefix(line, manifestPackageLine)
				if manifest.packageName == "" || strings.TrimSpace(manifest.packageName) != manifest.packageName {
					return shardManifest{}, fmt.Errorf("manifest %s row %d has malformed package header", manifestPath, row)
				}
				packageRow = row
			case strings.HasPrefix(line, manifestShardsLine):
				if shardsRow != 0 {
					return shardManifest{}, fmt.Errorf("manifest %s row %d duplicates shard-count header from row %d", manifestPath, row, shardsRow)
				}
				value := strings.TrimPrefix(line, manifestShardsLine)
				shardCount, err = strconv.Atoi(value)
				if err != nil || shardCount != expectedShardCount {
					return shardManifest{}, fmt.Errorf("manifest %s row %d has malformed shard count %q: want %d", manifestPath, row, value, expectedShardCount)
				}
				for index := 1; index <= shardCount; index++ {
					manifest.namesByShard[fmt.Sprintf("ui-%d", index)] = nil
				}
				shardsRow = row
			}
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 2 || fields[0] == "" || fields[1] == "" || strings.TrimSpace(fields[0]) != fields[0] || strings.TrimSpace(fields[1]) != fields[1] {
			return shardManifest{}, fmt.Errorf("manifest %s row %d is malformed: want test-name<TAB>shard", manifestPath, row)
		}
		if shardsRow == 0 {
			return shardManifest{}, fmt.Errorf("manifest %s row %d is malformed: assignment precedes shard-count header", manifestPath, row)
		}
		name, shardName := fields[0], fields[1]
		if name == "TestMain" || !isTopLevelRunnable(name) {
			return shardManifest{}, fmt.Errorf("manifest %s row %d has malformed top-level runnable name %q", manifestPath, row, name)
		}
		if firstRow, exists := manifest.assignmentRow[name]; exists {
			return shardManifest{}, fmt.Errorf("manifest %s row %d duplicates test %s assigned at row %d", manifestPath, row, name, firstRow)
		}
		if _, exists := manifest.namesByShard[shardName]; !exists {
			return shardManifest{}, fmt.Errorf("manifest %s row %d assigns test %s to unknown shard %s", manifestPath, row, name, shardName)
		}
		manifest.assignment[name] = shardName
		manifest.assignmentRow[name] = row
		manifest.namesByShard[shardName] = append(manifest.namesByShard[shardName], name)
	}
	if err := scanner.Err(); err != nil {
		return shardManifest{}, fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}
	if row == 0 {
		return shardManifest{}, fmt.Errorf("manifest %s is empty", manifestPath)
	}
	if packageRow == 0 {
		return shardManifest{}, fmt.Errorf("manifest %s is missing package header", manifestPath)
	}
	if shardsRow == 0 {
		return shardManifest{}, fmt.Errorf("manifest %s is missing shard-count header", manifestPath)
	}
	for index := 1; index <= shardCount; index++ {
		shardName := fmt.Sprintf("ui-%d", index)
		if len(manifest.namesByShard[shardName]) == 0 {
			return shardManifest{}, fmt.Errorf("manifest shard %s is empty", shardName)
		}
	}
	return manifest, nil
}

func writePlan(out io.Writer, summaries []streamSummary, packageName string, shardCount int, baselineSHA, baselineRun string) error {
	resolvedPackage, err := resolvePackage(packageName, summaries[0])
	if err != nil {
		return err
	}
	packageName = resolvedPackage

	weighted := make([]weightedTest, 0)
	for key := range summaries[0].tests {
		if key.packageName != packageName {
			continue
		}
		durations := make([]time.Duration, len(summaries))
		for index, summary := range summaries {
			result := summary.tests[key]
			if result.action != "pass" {
				return fmt.Errorf("run %d: %s/%s ended with %s; planning requires pass", index+1, key.packageName, key.name, result.action)
			}
			durations[index] = result.elapsed
		}
		weighted = append(weighted, weightedTest{name: key.name, weight: medianDuration(durations)})
	}
	if len(weighted) == 0 {
		return fmt.Errorf("package %s has no terminal top-level tests, fuzz targets, or examples", packageName)
	}
	sort.Slice(weighted, func(i, j int) bool {
		if weighted[i].weight != weighted[j].weight {
			return weighted[i].weight > weighted[j].weight
		}
		return weighted[i].name < weighted[j].name
	})

	shards := make([]shardPlan, shardCount)
	for index := range shards {
		shards[index].name = fmt.Sprintf("ui-%d", index+1)
	}
	for _, test := range weighted {
		target := 0
		for index := 1; index < len(shards); index++ {
			if shards[index].load < shards[target].load {
				target = index
			}
		}
		shards[target].tests = append(shards[target].tests, test.name)
		shards[target].load += test.weight
	}

	header := []string{
		"# PicFetch test shard manifest v1",
		"# package: " + packageName,
		fmt.Sprintf("# shards: %d", shardCount),
		"# baseline-sha: " + baselineSHA,
		"# baseline-run: " + baselineRun,
		"# baseline-attempts: 1,2,3",
	}
	for _, line := range header {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	for index := range shards {
		sort.Strings(shards[index].tests)
		if _, err := fmt.Fprintf(out, "# %s: %d entries, %.3fs median-weight sum\n", shards[index].name, len(shards[index].tests), durationSeconds(shards[index].load)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out, "# name\tshard"); err != nil {
		return err
	}
	for _, shard := range shards {
		for _, name := range shard.tests {
			if _, err := fmt.Fprintf(out, "%s\t%s\n", name, shard.name); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolvePackage(requested string, summary streamSummary) (string, error) {
	candidates := make(map[string]struct{})
	for key := range summary.tests {
		if key.packageName == requested {
			return requested, nil
		}
		candidates[key.packageName] = struct{}{}
	}
	if !strings.HasPrefix(requested, "./") {
		return "", fmt.Errorf("package %s is absent from the event inventory", requested)
	}

	relative := strings.TrimPrefix(path.Clean(requested), "./")
	if relative == "." || relative == "" || strings.Contains(relative, "...") {
		return "", fmt.Errorf("package %s must name one exact package", requested)
	}
	matches := make([]string, 0)
	for candidate := range candidates {
		if candidate == relative || strings.HasSuffix(candidate, "/"+relative) {
			matches = append(matches, candidate)
		}
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return "", fmt.Errorf("package %s is absent from the event inventory", requested)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("package %s is ambiguous in the event inventory: %s", requested, strings.Join(matches, ", "))
	}
	return matches[0], nil
}

func sortTestKeys(names []testKey) {
	sort.Slice(names, func(i, j int) bool {
		if names[i].packageName != names[j].packageName {
			return names[i].packageName < names[j].packageName
		}
		return names[i].name < names[j].name
	})
}

func writeResultRow(out io.Writer, prefix []string, values []terminal) error {
	fields := append([]string(nil), prefix...)
	durations := make([]time.Duration, len(values))
	for index, value := range values {
		fields = append(fields, fmt.Sprintf("run-%d=%s:%.3f", index+1, value.action, durationSeconds(value.elapsed)))
		durations[index] = value.elapsed
	}
	fields = append(fields, fmt.Sprintf("median=%.3f", durationSeconds(medianDuration(durations))))
	_, err := fmt.Fprintln(out, strings.Join(fields, "\t"))
	return err
}

func elapsedDuration(seconds float64) (time.Duration, error) {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return 0, fmt.Errorf("must be a finite non-negative number, got %g", seconds)
	}
	if seconds > float64(math.MaxInt64/int64(time.Second)) {
		return 0, fmt.Errorf("duration %g seconds is too large", seconds)
	}
	return time.Duration(math.Round(seconds * float64(time.Second))), nil
}

func durationSeconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Second)
}

func medianDuration(values []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), values...)
	slices.Sort(sorted)
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return sorted[middle-1] + (sorted[middle]-sorted[middle-1])/2
}
