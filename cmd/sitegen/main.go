package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frathe/picfetch/internal/sitegen"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: sitegen <build>")
	}
	switch args[0] {
	case "build":
		return runBuild(args[1:])
	case "check":
		return runCheck(args[1:])
	case "translate":
		return runTranslate(args[1:])
	case "update":
		return runUpdate(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runUpdate(args []string) error {
	flags := flag.NewFlagSet("update", flag.ContinueOnError)
	options := sitegen.BuildOptions{}
	var locales, formats, endpoint, environmentFile, nodeCommand, validatorPath string
	flags.StringVar(&options.SourcePath, "source", environmentOr("SITE_SOURCE", "website.md"), "English Markdown source")
	flags.StringVar(&options.TemplatesPath, "templates", environmentOr("SITE_TEMPLATES", "site/templates"), "template directory")
	flags.StringVar(&options.TranslationsPath, "translations", environmentOr("SITE_TRANSLATIONS", "site/translations/de.json"), "German translation cache")
	flags.StringVar(&options.OutputPath, "output", environmentOr("SITE_OUTPUT_DIR", "docs"), "generated output directory")
	flags.StringVar(&locales, "locales", environmentOr("SITE_LOCALES", "en,de"), "comma-separated locales")
	flags.StringVar(&formats, "formats", environmentOr("SITE_FORMATS", "regular,amp"), "comma-separated formats")
	flags.StringVar(&endpoint, "endpoint", environmentOr("DEEPL_API_URL", ""), "DeepL API base URL")
	flags.StringVar(&environmentFile, "env-file", environmentOr("DEEPL_ENV_FILE", ".env.local"), "ignored local environment file")
	flags.StringVar(&nodeCommand, "node", environmentOr("NODE", "node"), "Node.js executable")
	flags.StringVar(&validatorPath, "amp-validator", environmentOr("AMP_VALIDATOR", "site/tools/validate-amp.cjs"), "pinned AMP validator wrapper")
	if err := flags.Parse(args); err != nil {
		return err
	}
	apiKey, err := deepLAPIKey(environmentFile)
	if err != nil {
		return err
	}
	options.Locales = splitList(locales)
	options.Formats = splitList(formats)
	return sitegen.Update(sitegen.UpdateOptions{
		Build:            options,
		Endpoint:         endpoint,
		APIKey:           apiKey,
		NodeCommand:      nodeCommand,
		AMPValidatorPath: validatorPath,
	})
}

func runCheck(args []string) error {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	options := sitegen.BuildOptions{}
	var locales, formats string
	flags.StringVar(&options.SourcePath, "source", environmentOr("SITE_SOURCE", "website.md"), "English Markdown source")
	flags.StringVar(&options.TemplatesPath, "templates", environmentOr("SITE_TEMPLATES", "site/templates"), "template directory")
	flags.StringVar(&options.TranslationsPath, "translations", environmentOr("SITE_TRANSLATIONS", "site/translations/de.json"), "German translation cache")
	flags.StringVar(&options.OutputPath, "output", environmentOr("SITE_OUTPUT_DIR", "docs"), "committed generated output directory")
	flags.StringVar(&locales, "locales", environmentOr("SITE_LOCALES", "en,de"), "comma-separated locales")
	flags.StringVar(&formats, "formats", environmentOr("SITE_FORMATS", "regular,amp"), "comma-separated formats")
	if err := flags.Parse(args); err != nil {
		return err
	}
	options.Locales = splitList(locales)
	options.Formats = splitList(formats)
	return sitegen.Check(options)
}

func runTranslate(args []string) error {
	flags := flag.NewFlagSet("translate", flag.ContinueOnError)
	var source, cache, endpoint, environmentFile string
	flags.StringVar(&source, "source", environmentOr("SITE_SOURCE", "website.md"), "English Markdown source")
	flags.StringVar(&cache, "cache", environmentOr("SITE_TRANSLATIONS", "site/translations/de.json"), "German translation cache")
	flags.StringVar(&endpoint, "endpoint", environmentOr("DEEPL_API_URL", ""), "DeepL API base URL")
	flags.StringVar(&environmentFile, "env-file", environmentOr("DEEPL_ENV_FILE", ".env.local"), "ignored local environment file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	apiKey, err := deepLAPIKey(environmentFile)
	if err != nil {
		return err
	}
	return sitegen.Translate(sitegen.TranslateOptions{
		SourcePath: source,
		CachePath:  cache,
		Endpoint:   endpoint,
		APIKey:     apiKey,
	})
}

func deepLAPIKey(environmentFile string) (string, error) {
	if value := strings.TrimSpace(os.Getenv("DEEPL_API_KEY")); value != "" {
		return value, nil
	}
	data, err := os.ReadFile(environmentFile)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read local environment file %q: %w", environmentFile, err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		name, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(name) != "DEEPL_API_KEY" {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		return strings.TrimSpace(value), nil
	}
	return "", nil
}

func runBuild(args []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	var source, templates, translations, output, locales, formats string
	flags.StringVar(&source, "source", environmentOr("SITE_SOURCE", "website.md"), "English Markdown source")
	flags.StringVar(&templates, "templates", environmentOr("SITE_TEMPLATES", "site/templates"), "template directory")
	flags.StringVar(&translations, "translations", environmentOr("SITE_TRANSLATIONS", "site/translations/de.json"), "German translation cache")
	flags.StringVar(&output, "output", environmentOr("SITE_OUTPUT_DIR", "docs"), "generated output directory")
	flags.StringVar(&locales, "locales", environmentOr("SITE_LOCALES", "en,de"), "comma-separated locales")
	flags.StringVar(&formats, "formats", environmentOr("SITE_FORMATS", "regular,amp"), "comma-separated formats")
	if err := flags.Parse(args); err != nil {
		return err
	}
	return sitegen.Build(sitegen.BuildOptions{
		SourcePath:       source,
		TemplatesPath:    templates,
		TranslationsPath: translations,
		OutputPath:       output,
		Locales:          splitList(locales),
		Formats:          splitList(formats),
	})
}

func environmentOr(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
