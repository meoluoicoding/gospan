// Package cli implements the Gospan command-line interface. Run takes the
// program arguments and writers and returns a process exit code, which keeps
// the entrypoint thin and the behavior testable.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/snzpet/gospan/internal/app"
	"github.com/snzpet/gospan/internal/config"
	"github.com/snzpet/gospan/internal/engine"
	"github.com/snzpet/gospan/internal/platform"
	"github.com/snzpet/gospan/internal/platform/host"
	"github.com/snzpet/gospan/internal/storage"
	"github.com/snzpet/gospan/internal/update"
	"github.com/snzpet/gospan/internal/vocabulary"
)

// Exit codes. They are stable so scripts can rely on them.
const (
	ExitOK         = 0
	ExitError      = 1
	ExitUsage      = 2
	ExitValidation = 3
)

// Run executes the CLI and returns an exit code.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	a := &app2{stdout: stdout, stderr: stderr, version: version}
	return a.run(args)
}

// app2 is the CLI runtime. (Named app2 to avoid confusion with the app package.)
type app2 struct {
	stdout  io.Writer
	stderr  io.Writer
	version string
}

func (a *app2) run(args []string) int {
	if len(args) == 0 {
		a.printUsage()
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		a.printUsage()
		return ExitOK
	case "version", "-v", "--version":
		fmt.Fprintln(a.stdout, a.version)
		return ExitOK
	case "update":
		return a.updateCmd(args[1:])
	case "config":
		return a.configCmd(args[1:])
	case "match":
		return a.matchCmd(args[1:])
	case "vocab":
		return a.vocabCmd(args[1:])
	case "doctor":
		return a.doctorCmd(args[1:])
	case "status":
		return a.statusCmd(args[1:])
	case "start":
		return a.startCmd(args[1:])
	case "stop":
		fmt.Fprintln(a.stderr, "stop: no background instance control is implemented in this build")
		return ExitError
	default:
		fmt.Fprintf(a.stderr, "unknown command %q\n", args[0])
		a.printUsage()
		return ExitUsage
	}
}

func (a *app2) printUsage() {
	fmt.Fprint(a.stdout, `gospan — Go text expansion engine

Usage:
  gospan <command> [flags]

Commands:
  start                     Run the engine with keyboard capture
  stop                      Stop a running instance (not implemented)
  status                    Show configuration and engine status
  doctor                    Check the environment and configuration

  config validate           Validate the configuration
  config check              Check whether the configuration is valid and loaded
  config path               Print the resolved configuration path
  config init               Create a starter configuration
  config reload             Reload a running instance (not implemented)

  update check              Check for a newer release on GitHub

  match list                List configured triggers
  match test <input>        Test whether input ends with a trigger
  match add <trigger> <replacement>   Add a rule to the YAML configuration
  match remove <trigger>    Remove a rule from the YAML configuration

  vocab list                List vocabulary entries
  vocab add <trigger> <replacement>
  vocab export <file>       Export vocabulary as tab-separated text

  update check              Check for a newer release on GitHub

  version                   Print the version
  help                      Show this help

Global flags for most commands:
  --config <path>           Configuration file (default: GOSPAN_CONFIG or user dir)
  --json                    Machine-readable output
  --verbose / --quiet       Adjust output verbosity
`)
}

// --- shared flags -----------------------------------------------------------

type commonFlags struct {
	fs         *flag.FlagSet
	configPath string
	json       bool
	verbose    bool
	quiet      bool
}

func (a *app2) common(name string) *commonFlags {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	c := &commonFlags{fs: fs}
	fs.StringVar(&c.configPath, "config", "", "path to configuration file")
	fs.BoolVar(&c.json, "json", false, "emit JSON output")
	fs.BoolVar(&c.verbose, "verbose", false, "verbose output")
	fs.BoolVar(&c.quiet, "quiet", false, "suppress non-essential output")
	return c
}

// parse parses args and reports a usage failure as ExitUsage.
func (c *commonFlags) parse(args []string) (int, bool) {
	if err := c.fs.Parse(args); err != nil {
		return ExitUsage, false
	}
	return ExitOK, true
}

// parseInterspersed parses flags that may appear before or after positional
// arguments. The standard flag package stops at the first non-flag argument,
// which would silently ignore flags placed after positionals; this helper keeps
// collecting positionals and re-parsing the remainder. It returns the
// positional arguments in order.
func (c *commonFlags) parseInterspersed(args []string) ([]string, int, bool) {
	var positional []string
	rest := args
	for len(rest) > 0 {
		if err := c.fs.Parse(rest); err != nil {
			return nil, ExitUsage, false
		}
		rest = c.fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		rest = rest[1:]
	}
	return positional, ExitOK, true
}

// loadService opens the application service or writes an error and returns an
// exit code. The second return is false when loading failed.
func (a *app2) loadService(c *commonFlags) (*app.Service, int, bool) {
	cfgPath := config.ResolvePath(c.configPath)
	svc, err := app.Open(cfgPath, nil)
	if err != nil {
		a.reportConfigError(err)
		return nil, a.exitFor(err), false
	}
	return svc, ExitOK, true
}

// reportConfigError prints a configuration error, using the JSON form when the
// caller requested it.
func (a *app2) reportConfigError(err error) {
	var verrs config.ValidationErrors
	if errors.As(err, &verrs) {
		fmt.Fprintln(a.stderr, verrs.Error())
		return
	}
	fmt.Fprintf(a.stderr, "%v\n", err)
}

func (a *app2) exitFor(err error) int {
	var verrs config.ValidationErrors
	if errors.As(err, &verrs) {
		return ExitValidation
	}
	return ExitError
}

// --- config -----------------------------------------------------------------

func (a *app2) configCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "config: expected a subcommand (validate, path, reload)")
		return ExitUsage
	}
	switch args[0] {
	case "init":
		return a.configInit(args[1:])
	case "validate":
		return a.configValidate(args[1:])
	case "check":
		return a.configCheck(args[1:])
	case "path":
		return a.configPath(args[1:])
	case "reload":
		fmt.Fprintln(a.stderr, "config reload: requires a running instance (not implemented in this build)")
		return ExitError
	default:
		fmt.Fprintf(a.stderr, "config: unknown subcommand %q\n", args[0])
		return ExitUsage
	}
}

// configInit writes a starter configuration, failing if one already exists
// unless --force is given.
func (a *app2) configInit(args []string) int {
	c := a.common("config init")
	var force bool
	c.fs.BoolVar(&force, "force", false, "overwrite an existing configuration")
	if code, ok := c.parse(args); !ok {
		return code
	}

	path := config.ResolvePath(c.configPath)
	if force {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(a.stderr, "%v\n", err)
			return ExitError
		}
	}
	if err := config.WriteStarter(path); err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)
		return ExitError
	}
	if !c.quiet {
		fmt.Fprintf(a.stdout, "created %s\n", path)
	}
	return ExitOK
}

func (a *app2) configValidate(args []string) int {
	c := a.common("config validate")
	if code, ok := c.parse(args); !ok {
		return code
	}

	cfgPath := config.ResolvePath(c.configPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		a.reportConfigError(err)
		return a.exitFor(err)
	}

	if c.json {
		a.writeJSON(map[string]any{
			"ok":       true,
			"path":     cfg.App.ConfigPath,
			"matches":  len(cfg.Matches),
			"triggers": activeTriggers(cfg),
		})
		return ExitOK
	}
	if !c.quiet {
		fmt.Fprintf(a.stdout, "ok: %s (%d matches)\n", cfg.App.ConfigPath, len(cfg.Matches))
	}
	return ExitOK
}

func (a *app2) configCheck(args []string) int {
	c := a.common("config check")
	if code, ok := c.parse(args); !ok {
		return code
	}

	cfgPath := config.ResolvePath(c.configPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		a.reportConfigError(err)
		return a.exitFor(err)
	}

	// If a service can be opened, report whether it is actually loaded.
	loaded := false
	var reloadErr error
	if svc, _, ok := a.loadService(c); ok {
		loaded = svc.ConfigLoaded()
		reloadErr = svc.LastReloadError()
	}

	if c.json {
		a.writeJSON(map[string]any{
			"path":         cfg.App.ConfigPath,
			"valid":        true,
			"loaded":       loaded,
			"triggers":     activeTriggers(cfg),
			"backend":      cfg.Storage.Backend,
			"reload_error": errString(reloadErr),
		})
		return ExitOK
	}

	fmt.Fprintf(a.stdout, "config  %s\n", cfg.App.ConfigPath)
	fmt.Fprintf(a.stdout, "valid   yes\n")
	fmt.Fprintf(a.stdout, "loaded  %s\n", yesNo(loaded))
	fmt.Fprintf(a.stdout, "triggers %d\n", activeTriggers(cfg))
	fmt.Fprintf(a.stdout, "storage %s\n", cfg.Storage.Backend)
	if reloadErr != nil {
		fmt.Fprintf(a.stderr, "last reload error: %v\n", reloadErr)
	}
	return ExitOK
}

func (a *app2) configPath(args []string) int {
	c := a.common("config path")
	if code, ok := c.parse(args); !ok {
		return code
	}
	cfgPath := config.ResolvePath(c.configPath)
	def := config.DefaultPath()

	info := map[string]any{
		"path":    cfgPath,
		"default": def,
		"env":     config.EnvConfig,
	}
	if _, err := os.Stat(cfgPath); err == nil {
		info["exists"] = true
	} else {
		info["exists"] = false
	}

	if c.json {
		a.writeJSON(info)
		return ExitOK
	}
	fmt.Fprintln(a.stdout, cfgPath)
	return ExitOK
}

// --- match ------------------------------------------------------------------

func (a *app2) matchCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "match: expected a subcommand (list, test)")
		return ExitUsage
	}
	switch args[0] {
	case "list":
		return a.matchList(args[1:])
	case "test":
		return a.matchTest(args[1:])
	case "add":
		return a.matchAdd(args[1:])
	case "remove":
		return a.matchRemove(args[1:])
	default:
		fmt.Fprintf(a.stderr, "match: unknown subcommand %q\n", args[0])
		return ExitUsage
	}
}

func (a *app2) matchList(args []string) int {
	c := a.common("match list")
	var app string
	c.fs.StringVar(&app, "app", "", "foreground app name for app-specific filtering")
	if code, ok := c.parse(args); !ok {
		return code
	}
	svc, code, ok := a.loadService(c)
	if !ok {
		return code
	}

	cfg := svc.Config()
	if c.json {
		var out []config.MatchConfig
		for _, m := range cfg.Matches {
			if app != "" && !engine.AllowedByApps(app, m.Apps) {
				continue
			}
			out = append(out, m)
		}
		a.writeJSON(out)
		return ExitOK
	}
	for _, m := range cfg.Matches {
		if app != "" && !engine.AllowedByApps(app, m.Apps) {
			continue
		}
		state := ""
		if !m.IsEnabled() {
			state = " (disabled)"
		}
		fmt.Fprintf(a.stdout, "%-16s → %s%s\n", m.Trigger, firstLine(m.Replace), state)
	}
	return ExitOK
}

func (a *app2) matchTest(args []string) int {
	c := a.common("match test")
	var app string
	c.fs.StringVar(&app, "app", "", "foreground app name for app-specific filtering")
	positional, code, ok := c.parseInterspersed(args)
	if !ok {
		return code
	}
	if len(positional) != 1 {
		fmt.Fprintln(a.stderr, "match test: expected exactly one input string")
		return ExitUsage
	}
	input := positional[0]

	svc, code, ok := a.loadService(c)
	if !ok {
		return code
	}

	m, snippet, matched, err := svc.TestSnippet(input, app)
	if err != nil {
		fmt.Fprintf(a.stderr, "render error: %v\n", err)
		return ExitError
	}

	if c.json {
		a.writeJSON(map[string]any{
			"input":    input,
			"matched":  matched,
			"trigger":  m.Trigger,
			"output":   snippet.Text,
			"cursor":   snippet.CursorOffset,
			"tabstops": snippet.TabStops,
		})
		return ExitOK
	}
	if !matched {
		fmt.Fprintf(a.stdout, "no match for %q\n", input)
		return ExitOK
	}
	out := snippet.Text
	if snippet.CursorOffset >= 0 && snippet.CursorOffset < len([]rune(out)) {
		r := []rune(out)
		out = string(r[:snippet.CursorOffset]) + "|" + string(r[snippet.CursorOffset:])
	}
	fmt.Fprintf(a.stdout, "%s → %s\n", m.Trigger, out)
	return ExitOK
}

func (a *app2) matchAdd(args []string) int {
	c := a.common("match add")
	var file, description string
	c.fs.StringVar(&file, "file", "", "target YAML file (default: the loaded configuration)")
	c.fs.StringVar(&description, "description", "", "optional rule description")
	positional, code, ok := c.parseInterspersed(args)
	if !ok {
		return code
	}
	if len(positional) != 2 {
		fmt.Fprintln(a.stderr, "match add: expected <trigger> <replacement>")
		return ExitUsage
	}
	trigger := positional[0]
	replacement := positional[1]

	root := config.ResolvePath(c.configPath)
	target, validateRoot := file, ""
	if target == "" {
		target = root
		validateRoot = root
	}

	editor := config.NewEditor(target, validateRoot)
	if err := editor.AddMatch(trigger, replacement, description); err != nil {
		a.reportConfigError(err)
		return a.exitFor(err)
	}
	if !c.quiet {
		fmt.Fprintf(a.stdout, "added %s → %s to %s\n", trigger, firstLine(replacement), editor.Path())
	}
	return ExitOK
}

func (a *app2) matchRemove(args []string) int {
	c := a.common("match remove")
	var file string
	c.fs.StringVar(&file, "file", "", "target YAML file (default: the loaded configuration)")
	positional, code, ok := c.parseInterspersed(args)
	if !ok {
		return code
	}
	if len(positional) != 1 {
		fmt.Fprintln(a.stderr, "match remove: expected <trigger>")
		return ExitUsage
	}
	trigger := positional[0]

	root := config.ResolvePath(c.configPath)

	// Without an explicit --file, locate the rule (including those defined in
	// included dictionaries) and remove it from the file that owns it.
	caseInsensitive := true
	target, validateRoot := file, ""
	if target != "" {
		validateRoot = root
	} else {
		target = root
		validateRoot = root
		if cfg, err := config.Load(root); err == nil {
			caseInsensitive = !cfg.Engine.CaseSensitive
			for _, m := range cfg.Matches {
				if m.Source == "" {
					continue
				}
				if triggerMatches(m.Trigger, trigger, caseInsensitive) {
					target = m.Source
					break
				}
			}
		}
	}

	editor := config.NewEditor(target, validateRoot)
	removed, err := editor.RemoveMatch(trigger, caseInsensitive)
	if err != nil {
		a.reportConfigError(err)
		return a.exitFor(err)
	}
	if !removed {
		fmt.Fprintf(a.stderr, "no trigger %q found in %s\n", trigger, editor.Path())
		return ExitError
	}
	if !c.quiet {
		fmt.Fprintf(a.stdout, "removed %s from %s\n", trigger, editor.Path())
	}
	return ExitOK
}

// --- vocab ------------------------------------------------------------------

func (a *app2) vocabCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "vocab: expected a subcommand (list, add, export)")
		return ExitUsage
	}
	switch args[0] {
	case "list":
		return a.vocabList(args[1:])
	case "add":
		return a.vocabAdd(args[1:])
	case "export":
		return a.vocabExport(args[1:])
	default:
		fmt.Fprintf(a.stderr, "vocab: unknown subcommand %q\n", args[0])
		return ExitUsage
	}
}

func (a *app2) openVocabulary(c *commonFlags) (*vocabulary.Service, int, bool) {
	svc, code, ok := a.loadService(c)
	if !ok {
		return nil, code, false
	}
	cfg := svc.Config()
	store, err := storage.Open(cfg.Storage.Backend, resolveStoragePath(cfg))
	if err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)
		return nil, ExitError, false
	}
	return vocabulary.NewService(store, cfg.Suggestions), ExitOK, true
}

func (a *app2) vocabList(args []string) int {
	c := a.common("vocab list")
	if code, ok := c.parse(args); !ok {
		return code
	}
	vs, code, ok := a.openVocabulary(c)
	if !ok {
		return code
	}
	defer vs.Close()

	entries, err := vs.List()
	if err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)
		return ExitError
	}
	if c.json {
		a.writeJSON(entries)
		return ExitOK
	}
	for _, e := range entries {
		fmt.Fprintf(a.stdout, "%-16s → %s\n", e.Trigger, firstLine(e.Expansion))
	}
	return ExitOK
}

func (a *app2) vocabAdd(args []string) int {
	c := a.common("vocab add")
	positional, code, ok := c.parseInterspersed(args)
	if !ok {
		return code
	}
	if len(positional) != 2 {
		fmt.Fprintln(a.stderr, "vocab add: expected <trigger> <replacement>")
		return ExitUsage
	}
	trigger := positional[0]
	replacement := positional[1]

	vs, code, ok := a.openVocabulary(c)
	if !ok {
		return code
	}
	defer vs.Close()

	if err := vs.Add(trigger, replacement, ""); err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)
		return ExitError
	}
	if !c.quiet {
		fmt.Fprintf(a.stdout, "added %s → %s\n", trigger, replacement)
	}
	return ExitOK
}

func (a *app2) vocabExport(args []string) int {
	c := a.common("vocab export")
	positional, code, ok := c.parseInterspersed(args)
	if !ok {
		return code
	}
	if len(positional) != 1 {
		fmt.Fprintln(a.stderr, "vocab export: expected an output file")
		return ExitUsage
	}
	out, err := os.Create(positional[0])
	if err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)
		return ExitError
	}
	defer out.Close()

	vs, code, ok := a.openVocabulary(c)
	if !ok {
		return code
	}
	defer vs.Close()

	if err := vs.Export(out); err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)
		return ExitError
	}
	if !c.quiet {
		fmt.Fprintf(a.stdout, "exported to %s\n", positional[0])
	}
	return ExitOK
}

// --- doctor / status / start ------------------------------------------------

func (a *app2) doctorCmd(args []string) int {
	c := a.common("doctor")
	if code, ok := c.parse(args); !ok {
		return code
	}

	cfgPath := config.ResolvePath(c.configPath)
	report := map[string]any{
		"version":   a.version,
		"go":        runtime.Version(),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"config":    cfgPath,
		"config_ok": false,
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		report["config_error"] = err.Error()
		if c.json {
			a.writeJSON(report)
		} else {
			fmt.Fprintf(a.stdout, "config: FAILED\n")
			a.reportConfigError(err)
		}
		return a.exitFor(err)
	}
	report["config_ok"] = true
	report["triggers"] = activeTriggers(cfg)
	report["storage_backend"] = cfg.Storage.Backend

	plat, perr := host.New()
	report["platform"] = runtime.GOOS
	_, isUnsupported := plat.(interface {
		Events() <-chan platform.KeyboardEvent
	})
	report["platform_available"] = perr == nil && isUnsupported

	if c.json {
		a.writeJSON(report)
		return ExitOK
	}

	fmt.Fprintf(a.stdout, "gospan doctor\n")
	fmt.Fprintf(a.stdout, "  version          %s\n", a.version)
	fmt.Fprintf(a.stdout, "  go               %s\n", runtime.Version())
	fmt.Fprintf(a.stdout, "  os/arch          %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(a.stdout, "  config           %s\n", cfgPath)
	fmt.Fprintf(a.stdout, "  config valid     yes\n")
	fmt.Fprintf(a.stdout, "  triggers         %d\n", activeTriggers(cfg))
	fmt.Fprintf(a.stdout, "  storage backend  %s\n", cfg.Storage.Backend)
	return ExitOK
}

func (a *app2) statusCmd(args []string) int {
	c := a.common("status")
	if code, ok := c.parse(args); !ok {
		return code
	}
	svc, code, ok := a.loadService(c)
	if !ok {
		return code
	}
	cfg := svc.Config()

	loaded := svc.ConfigLoaded()
	reloadErr := svc.LastReloadError()

	if c.json {
		a.writeJSON(map[string]any{
			"config":        cfg.App.ConfigPath,
			"config_loaded": loaded,
			"triggers":      svc.TriggerCount(),
			"running":       false,
			"reload_error":  errString(reloadErr),
		})
		return ExitOK
	}
	fmt.Fprintf(a.stdout, "config        %s\n", cfg.App.ConfigPath)
	fmt.Fprintf(a.stdout, "config loaded %s\n", yesNo(loaded))
	fmt.Fprintf(a.stdout, "triggers      %d\n", svc.TriggerCount())
	fmt.Fprintf(a.stdout, "running       no (foreground process not managed by this command)\n")
	if reloadErr != nil {
		fmt.Fprintf(a.stderr, "last reload error: %v\n", reloadErr)
	}
	return ExitOK
}

func (a *app2) startCmd(args []string) int {
	c := a.common("start")
	if code, ok := c.parse(args); !ok {
		return code
	}
	svc, code, ok := a.loadService(c)
	if !ok {
		return code
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := svc.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(a.stderr, "start: %v\n", err)
		return ExitError
	}
	return ExitOK
}

// --- helpers ----------------------------------------------------------------

func activeTriggers(cfg *config.Config) int {
	n := 0
	for _, m := range cfg.Matches {
		if m.IsEnabled() {
			n++
		}
	}
	return n
}

func resolveStoragePath(cfg *config.Config) string {
	p := cfg.Storage.Path
	if filepath.IsAbs(p) {
		return p
	}
	dir := filepath.Dir(cfg.App.ConfigPath)
	if dir == "" || dir == "." {
		return p
	}
	return filepath.Join(dir, p)
}

func triggerMatches(a, b string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func (a *app2) updateCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "update: expected a subcommand (check)")
		return ExitUsage
	}
	switch args[0] {
	case "check":
		return a.updateCheck(args[1:])
	default:
		fmt.Fprintf(a.stderr, "update: unknown subcommand %q\n", args[0])
		return ExitUsage
	}
}

func (a *app2) updateCheck(args []string) int {
	c := a.common("update check")
	if code, ok := c.parse(args); !ok {
		return code
	}

	const repo = "meoluoicoding/gospan"
	rel, err := update.CheckLatest(repo, 10*time.Second)
	if err != nil {
		fmt.Fprintf(a.stderr, "update check failed: %v\n", err)
		return ExitError
	}

	if c.json {
		a.writeJSON(map[string]any{
			"current": a.version,
			"latest":  rel.TagName,
			"newer":   update.IsNewer(a.version, rel.TagName),
			"url":     rel.HTMLURL,
		})
		return ExitOK
	}

	fmt.Fprintf(a.stdout, "current version: %s\n", a.version)
	fmt.Fprintf(a.stdout, "latest release:  %s\n", rel.TagName)
	if update.IsNewer(a.version, rel.TagName) {
		fmt.Fprintf(a.stdout, "A newer version is available: %s\n", rel.HTMLURL)
	} else {
		fmt.Fprintln(a.stdout, "You are up to date.")
	}
	return ExitOK
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + "…"
	}
	return s
}

func (a *app2) writeJSON(v any) {
	enc := json.NewEncoder(a.stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
