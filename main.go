package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/whitehatsec/glicense/output"

	"github.com/fatih/color"
	"github.com/google/go-github/v18/github"
	"golang.org/x/oauth2"

	"github.com/whitehatsec/glicense/config"
	"github.com/whitehatsec/glicense/license"
	githubFinder "github.com/whitehatsec/glicense/license/github"
	"github.com/whitehatsec/glicense/license/golang"
	"github.com/whitehatsec/glicense/license/gopkg"
	"github.com/whitehatsec/glicense/license/mapper"
	"github.com/whitehatsec/glicense/license/resolver"
	"github.com/whitehatsec/glicense/module"
	"golang.org/x/mod/modfile"
)

const (
	EnvGitHubToken = "GITHUB_TOKEN"
	goMod          = "go.mod"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	termOut := &output.TermOutput{Out: os.Stdout}

	var flagLicense bool
	var flagOutXLSX string
	var flagOutCSV string
	var flagIndirect bool
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.BoolVar(&flagIndirect, "indirect", false,
		"include indirect dependencies in report")
	flags.BoolVar(&flagLicense, "license", true,
		"look up and verify license. If false, dependencies are\n"+
			"printed without licenses.")
	flags.BoolVar(&termOut.Plain, "plain", false, "plain terminal output, no colors or live updates")
	flags.BoolVar(&termOut.Verbose, "verbose", false, "additional logging to terminal, requires -plain")
	flags.StringVar(&flagOutXLSX, "out-xlsx", "",
		"save report in Excel XLSX format to the given path")
	flags.StringVar(&flagOutCSV, "out-csv", "",
		"save report in CSV format to the given path")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		printHelp(flags)
		return 1
	}

	args := flags.Args()
	if len(args) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, color.RedString(
			"❗️ Path to directory to analyze expected.\n\n"))
		printHelp(flags)
		return 1
	}

	// Determine the exe path and parse the configuration if given.
	var cfg config.Config
	dirPaths := args[:1]
	if len(args) > 1 {
		dirPaths = args[1:]

		c, err := config.ParseFile(args[0])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, color.RedString(fmt.Sprintf(
				"❗️ Error parsing configuration:\n\n%s\n", err)))
			return 1
		}

		// Store the config and set it on the output
		cfg = *c
	}

	allMods := map[module.Module]struct{}{}
	for _, dirPath := range dirPaths {
		// Read the dependencies from the binary itself
		bts, err := os.ReadFile(filepath.Join(dirPath, goMod))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, color.RedString(fmt.Sprintf(
				"❗️ Error reading %q: %s\n", args[0], err)))
			return 1
		}

		file, err := modfile.Parse(goMod, bts, nil)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, color.RedString(fmt.Sprintf(
				"❗️ Error reading %q: %s\n", args[0], err)))
			return 1
		}

		for _, require := range file.Require {
			if !flagIndirect && require.Indirect {
				continue
			}

			mod := toModule(require, require.Indirect)
			allMods[mod] = struct{}{}
		}
	}

	mods := make([]module.Module, 0, len(allMods))
	for mod := range allMods {
		mods = append(mods, mod)
	}

	// Complete terminal output setup
	termOut.Config = &cfg
	termOut.Modules = mods

	// Setup the outputs
	out := &output.MultiOutput{Outputs: []output.Output{termOut}}
	if flagOutXLSX != "" {
		out.Outputs = append(out.Outputs, &output.XLSXOutput{
			Path:   flagOutXLSX,
			Config: &cfg,
		})
	}

	if flagOutCSV != "" {
		out.Outputs = append(out.Outputs, &output.CSVOutput{
			Path:   flagOutCSV,
			Config: &cfg,
		})
	}

	// Setup a context. We don't connect this to an interrupt signal or
	// anything since we just exit immediately on interrupt. No cleanup
	// necessary.
	ctx := context.Background()

	// Auth with GitHub if available
	var githubClient *http.Client
	if v := os.Getenv(EnvGitHubToken); v != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: v})
		githubClient = oauth2.NewClient(ctx, ts)
	}

	// Build our translators and license finders
	ts := []license.Translator{
		&mapper.Translator{Map: cfg.Translate},
		&resolver.Translator{},
		&golang.Translator{},
		&gopkg.Translator{},
	}
	var fs []license.Finder
	if flagLicense {
		fs = []license.Finder{
			&mapper.Finder{Map: cfg.Override},
			&githubFinder.RepoAPI{
				Client: github.NewClient(githubClient),
			},
		}
	}

	// Kick off all the license lookups.
	var wg sync.WaitGroup
	sem := NewSemaphore(5)
	for _, m := range mods {
		wg.Add(1)
		go func(m module.Module) {
			defer wg.Done()

			// Acquire a semaphore so that we can limit concurrency
			sem.Acquire()
			defer sem.Release()

			// Build the context
			ctx := license.StatusWithContext(ctx, output.StatusListener(out, &m))

			// Lookup
			out.Start(&m)

			// We first try the untranslated version. If we can detect
			// a license then take that. Otherwise, we translate.
			lic, err := license.Find(ctx, m, fs)
			if lic == nil || err != nil {
				lic, err = license.Find(ctx, license.Translate(ctx, m, ts), fs)
			}
			out.Finish(&m, lic, err)
		}(m)
	}

	// Wait for all lookups to complete
	wg.Wait()

	// Close the output
	if err := out.Close(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, color.RedString(fmt.Sprintf(
			"❗️ Error: %s\n", err)))
		return 1
	}

	return termOut.ExitCode()
}

func toModule(require *modfile.Require, indirect bool) module.Module {
	return module.Module{
		Path:     require.Mod.Path,
		Version:  require.Mod.Version,
		Indirect: indirect,
	}
}

func printHelp(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, strings.TrimSpace(help)+"\n\n", os.Args[0])
	fs.PrintDefaults()
}

const help = `
glicense analyzes the dependencies of a binary compiled from Go.

Usage: %[1]s [flags] [MODULE_DIR]
Usage: %[1]s [flags] [CONFIG] [MODULE_DIR]

One or two arguments can be given: a go module directory by itself which will 
output all the licenses of dependencies, or a configuration file and a module
directory which also notes which licenses are allowed among other settings.

For full help text, see the README in the GitHub repository:
http://github.com/whitehatsec/glicense

Flags:

`
