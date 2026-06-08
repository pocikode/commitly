package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	xterm "github.com/charmbracelet/x/term"
	"github.com/pocikode/opencommit/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get or set configuration values",
		Long:  "Manage OpenCommit-Go configuration. With no subcommand, opens the interactive profile manager.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigManager(cmd.InOrStdin(), cmd.OutOrStdout(), flagConfig)
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "get <KEY> [<KEY> ...]",
			Short: "Print one or more configuration values",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runConfigGet(cmd.OutOrStdout(), flagConfig, args)
			},
		},
		&cobra.Command{
			Use:   "set <KEY>=<VALUE> [<KEY>=<VALUE> ...]",
			Short: "Set one or more configuration values",
			Args:  cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runConfigSet(cmd.OutOrStdout(), flagConfig, args)
			},
		},
		&cobra.Command{
			Use:     "profiles",
			Aliases: []string{"list"},
			Short:   "List saved provider/model profiles",
			Args:    cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runConfigProfiles(cmd.OutOrStdout(), flagConfig)
			},
		},
		&cobra.Command{
			Use:   "use <NAME>",
			Short: "Switch the active provider/model profile",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runConfigUse(cmd.OutOrStdout(), flagConfig, args[0])
			},
		},
	)

	return cmd
}

// runConfigGet prints the effective value for each requested key. The api_key
// value is redacted.
func runConfigGet(out io.Writer, configPath string, keys []string) error {
	cfg, _, err := config.Resolve(config.Options{ConfigPath: configPath})
	if err != nil {
		return err
	}
	for _, key := range keys {
		val, err := config.Get(&cfg, key)
		if err != nil {
			return err
		}
		if key == "api_key" {
			val = redact(val)
		}
		fmt.Fprintf(out, "%s=%s\n", key, val)
	}
	return nil
}

// runConfigSet parses KEY=VALUE pairs, validates each, and persists them to the
// global config file.
func runConfigSet(out io.Writer, configPath string, pairs []string) error {
	path, err := config.GlobalPath(configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return fmt.Errorf("invalid argument %q: expected KEY=VALUE", pair)
		}
		key = strings.TrimSpace(key)
		if err := config.Set(&cfg, key, value); err != nil {
			return err
		}
	}

	if err := config.Save(cfg, path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Saved %d value(s) to %s\n", len(pairs), path)
	return nil
}

// runConfigProfiles lists saved profiles, marking the active one, with a short
// provider/model summary. API keys are not printed.
func runConfigProfiles(out io.Writer, configPath string) error {
	path, err := config.GlobalPath(configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		fmt.Fprintln(out, "No profiles saved. Run `oco config` to create one.")
		return nil
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := cfg.Profiles[name]
		marker := "  "
		if name == cfg.ActiveProfile {
			marker = "* "
		}
		fmt.Fprintf(out, "%s%s\t%s/%s\n", marker, name, p.AIProvider, p.Model)
	}
	return nil
}

// runConfigUse switches the active profile to name and mirrors it onto the
// top-level provider fields, then persists.
func runConfigUse(out io.Writer, configPath, name string) error {
	path, err := config.GlobalPath(configPath)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	profile, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("unknown profile %q: run `oco config profiles` to list them", name)
	}
	cfg.ActiveProfile = name
	config.ApplyProfile(&cfg, profile)
	if err := config.Save(cfg, path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Active profile is now %q (%s/%s)\n", name, profile.AIProvider, profile.Model)
	return nil
}

// wizardValues holds the settings collected by the profile wizard. name is the
// profile the entry is saved under.
type wizardValues struct {
	name     string
	provider string
	ptype    string
	apiURL   string
	apiKey   string
	model    string
}

// Profile-manager actions returned by the list view.
const (
	actionQuit   = "quit"
	actionUse    = "use"
	actionCreate = "create"
	actionEdit   = "edit"
	actionDelete = "delete"
)

// runConfigManager is the interactive profile manager shown by bare
// `oco config`: it lists saved profiles and lets the user add, edit, delete, or
// activate one. On a real terminal it uses a bubbletea key-driven list; for
// piped/test input it falls back to a line-based menu. Both share one reader so
// the flow stays scriptable and unit-testable.
func runConfigManager(in io.Reader, out io.Writer, configPath string) error {
	tty := inputIsTerminal(in)
	var r *bufio.Reader
	if !tty {
		r = bufio.NewReader(in)
	}

	for {
		path, err := config.GlobalPath(configPath)
		if err != nil {
			return err
		}
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}
		names := sortedProfileNames(cfg)

		var action, sel string
		if tty {
			action, sel, err = runProfileTUI(in, out, cfg, names)
		} else {
			action, sel, err = profileMenuText(r, out, cfg, names)
		}
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		switch action {
		case actionQuit:
			return nil

		case actionUse:
			if sel == "" {
				continue
			}
			cfg.ActiveProfile = sel
			config.ApplyProfile(&cfg, cfg.Profiles[sel])
			if err := config.Save(cfg, path); err != nil {
				return err
			}
			fmt.Fprintf(out, "Active profile is now %q\n", sel)
			return nil

		case actionDelete:
			if sel == "" {
				continue
			}
			delete(cfg.Profiles, sel)
			if cfg.ActiveProfile == sel {
				cfg.ActiveProfile = ""
			}
			if err := config.Save(cfg, path); err != nil {
				return err
			}
			fmt.Fprintf(out, "Deleted profile %q\n", sel)

		case actionCreate:
			if err := collectAndSaveProfile(tty, in, r, out, &cfg, path, "", true); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}

		case actionEdit:
			if sel == "" {
				continue
			}
			if err := collectAndSaveProfile(tty, in, r, out, &cfg, path, sel, false); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return err
			}
		}
	}
}

// collectAndSaveProfile runs the field wizard (seeded from an existing profile
// when name is non-empty) and writes the result back to cfg + disk. The first
// profile created becomes active automatically.
func collectAndSaveProfile(tty bool, in io.Reader, r *bufio.Reader, out io.Writer, cfg *config.Config, path, name string, askName bool) error {
	base := cfg.Profiles[name] // zero value when creating
	vals := wizardValues{
		name:     defaultStr(name, "default"),
		provider: defaultStr(base.AIProvider, config.KnownProviders[0]),
		ptype:    defaultStr(base.ProviderType, config.ProviderTypeOpenAICompatible),
		apiURL:   base.APIURL,
		apiKey:   base.APIKey,
		model:    base.Model,
	}

	var err error
	if tty {
		err = collectWizardHuh(in, out, &vals, askName)
	} else {
		err = collectWizardText(r, out, cfg, &vals, askName)
	}
	if err != nil {
		return err
	}

	final := strings.TrimSpace(vals.name)
	if final == "" {
		final = "default"
	}
	profile := config.Profile{
		AIProvider:   vals.provider,
		ProviderType: vals.ptype,
		APIURL:       vals.apiURL,
		APIKey:       vals.apiKey,
		Model:        vals.model,
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.Profile{}
	}
	cfg.Profiles[final] = profile
	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = final
	}
	// Mirror the active profile onto the top-level provider fields so plain
	// `config get` and direct file reads see the effective values.
	if cfg.ActiveProfile == final {
		config.ApplyProfile(cfg, profile)
	}
	if err := config.Save(*cfg, path); err != nil {
		return err
	}
	fmt.Fprintf(out, "Saved profile %q\n", final)
	return nil
}

// sortedProfileNames returns the profile names in stable alphabetical order.
func sortedProfileNames(cfg config.Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// profileMenuText renders the line-based profile menu and parses one command:
// `a` add, `e N` edit, `d N` delete, `N` use the Nth profile, blank/`q` quit.
func profileMenuText(r *bufio.Reader, out io.Writer, cfg config.Config, names []string) (string, string, error) {
	if len(names) == 0 {
		fmt.Fprintln(out, "No profiles yet.")
	} else {
		fmt.Fprintln(out, "Profiles:")
		for i, n := range names {
			marker := "  "
			if n == cfg.ActiveProfile {
				marker = "* "
			}
			p := cfg.Profiles[n]
			fmt.Fprintf(out, "  %s%d) %s (%s/%s)\n", marker, i+1, n, p.AIProvider, p.Model)
		}
	}
	fmt.Fprint(out, "[a]dd  [e N]edit  [d N]delete  [N]use  [q]uit: ")

	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" || line == "q" {
		return actionQuit, "", nil
	}
	fields := strings.Fields(line)
	pick := func(s string) string {
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(names) {
			return names[n-1]
		}
		return ""
	}
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	switch fields[0] {
	case "a":
		return actionCreate, "", nil
	case "e":
		return actionEdit, pick(arg), nil
	case "d":
		return actionDelete, pick(arg), nil
	default:
		return actionUse, pick(fields[0]), nil
	}
}

// profileTUIModel is the bubbletea list for the interactive profile manager.
type profileTUIModel struct {
	names  []string
	cfg    config.Config
	cursor int
	action string
	sel    string
}

func (m profileTUIModel) Init() tea.Cmd { return nil }

func (m profileTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.names)-1 {
			m.cursor++
		}
	case "a":
		m.action = actionCreate
		return m, tea.Quit
	case "e":
		if len(m.names) > 0 {
			m.action, m.sel = actionEdit, m.names[m.cursor]
		}
		return m, tea.Quit
	case "d":
		if len(m.names) > 0 {
			m.action, m.sel = actionDelete, m.names[m.cursor]
		}
		return m, tea.Quit
	case "enter":
		if len(m.names) > 0 {
			m.action, m.sel = actionUse, m.names[m.cursor]
		}
		return m, tea.Quit
	case "q", "esc", "ctrl+c":
		m.action = actionQuit
		return m, tea.Quit
	}
	return m, nil
}

// Profile-manager TUI styles. lipgloss emits no escape codes on a non-TTY
// writer, so the rendered text degrades to plain ASCII in tests/pipes.
var (
	tuiTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("213")).
			MarginBottom(1)
	tuiBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 2)
	tuiSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("63"))
	tuiActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	tuiRow      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	tuiEmpty    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	tuiHelpKey  = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
	tuiHelpText = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func (m profileTUIModel) View() string {
	var rows strings.Builder
	if len(m.names) == 0 {
		rows.WriteString(tuiEmpty.Render("  none yet — press a to add"))
	}

	// First pass: build each row as plain text and track the widest so the
	// selection highlight can fill the whole row.
	plains := make([]string, len(m.names))
	width := 0
	for i, n := range m.names {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		marker := "  "
		if n == m.cfg.ActiveProfile {
			marker = "* "
		}
		p := m.cfg.Profiles[n]
		plains[i] = fmt.Sprintf("%s%s%s (%s/%s)", cursor, marker, n, p.AIProvider, p.Model)
		if w := lipgloss.Width(plains[i]); w > width {
			width = w
		}
	}

	for i, n := range m.names {
		// A single style per row keeps the visible characters contiguous (no
		// ANSI codes mid-text), which the substring tests rely on.
		style := tuiRow
		if n == m.cfg.ActiveProfile {
			style = tuiActive
		}
		if i == m.cursor {
			style = tuiSelected
		}
		if i > 0 {
			rows.WriteByte('\n')
		}
		rows.WriteString(style.Width(width).Render(plains[i]))
	}

	help := tuiHelpText.Render("  ") +
		tuiHelpKey.Render("[a]dd") + tuiHelpText.Render("  ") +
		tuiHelpKey.Render("[e]dit") + tuiHelpText.Render("  ") +
		tuiHelpKey.Render("[d]elete") + tuiHelpText.Render("  ") +
		tuiHelpKey.Render("[enter]use") + tuiHelpText.Render("  ") +
		tuiHelpKey.Render("[q]uit")

	return tuiTitle.Render("✨ OpenCommit Profiles") + "\n" +
		tuiBox.Render(rows.String()) + "\n" +
		help + "\n"
}

// runProfileTUI runs the bubbletea list and returns the chosen action and the
// selected profile name.
func runProfileTUI(in io.Reader, out io.Writer, cfg config.Config, names []string) (string, string, error) {
	res, err := tea.NewProgram(
		profileTUIModel{names: names, cfg: cfg},
		tea.WithInput(in),
		tea.WithOutput(out),
	).Run()
	if err != nil {
		return "", "", err
	}
	m := res.(profileTUIModel)
	if m.action == "" {
		return actionQuit, "", nil
	}
	return m.action, m.sel, nil
}

// collectWizardHuh drives the rich huh UI on a real terminal. Each field is its
// own group so the form advances one step at a time (stepper), instead of
// showing every field at once. The profile-name step is shown only when
// askName is set (creating, not editing).
func collectWizardHuh(in io.Reader, out io.Writer, vals *wizardValues, askName bool) error {
	groups := []*huh.Group{}
	if askName {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().
				Title("Profile name").
				Description("a label for this provider/model setup").
				Value(&vals.name),
		))
	}
	groups = append(groups,
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI provider").
				Options(stringOptions(config.KnownProviders)...).
				Value(&vals.provider),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Provider type").
				Options(
					huh.NewOption("OpenAI-compatible", config.ProviderTypeOpenAICompatible),
					huh.NewOption("Anthropic-compatible", config.ProviderTypeAnthropicCompatible),
				).
				Value(&vals.ptype),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("API base URL").
				Description("blank uses the provider default").
				Value(&vals.apiURL),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("API key").
				EchoMode(huh.EchoModePassword).
				Value(&vals.apiKey),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Model").
				Value(&vals.model),
		),
	)
	return huh.NewForm(groups...).WithInput(in).WithOutput(out).Run()
}

// collectWizardText drives a line-based selector for non-TTY input, sharing the
// caller's bufio.Reader so no buffered bytes are lost across prompts. The
// profile-name step is shown only when askName is set (creating); when it names
// an existing profile, that profile's values prefill the remaining prompts.
func collectWizardText(r *bufio.Reader, out io.Writer, cfg *config.Config, vals *wizardValues, askName bool) error {
	if askName {
		vals.name = askLine(r, out, "Profile name", vals.name)
		if p, ok := cfg.Profiles[strings.TrimSpace(vals.name)]; ok {
			vals.provider = defaultStr(p.AIProvider, vals.provider)
			vals.ptype = defaultStr(p.ProviderType, vals.ptype)
			vals.apiURL = defaultStr(p.APIURL, vals.apiURL)
			vals.apiKey = defaultStr(p.APIKey, vals.apiKey)
			vals.model = defaultStr(p.Model, vals.model)
		}
	}

	vals.provider = selectLine(r, out, "AI provider", config.KnownProviders, vals.provider)
	vals.ptype = selectLine(r, out, "Provider type",
		[]string{config.ProviderTypeOpenAICompatible, config.ProviderTypeAnthropicCompatible}, vals.ptype)
	vals.apiURL = askLine(r, out, "API base URL (blank uses provider default)", vals.apiURL)
	vals.apiKey = askLine(r, out, "API key", vals.apiKey)
	vals.model = askLine(r, out, "Model", vals.model)
	return nil
}

// selectLine prints a numbered menu and returns the chosen option. Accepts the
// 1-based index or the literal value; a blank line keeps def.
func selectLine(r *bufio.Reader, out io.Writer, label string, options []string, def string) string {
	fmt.Fprintf(out, "%s:\n", label)
	for i, o := range options {
		marker := "  "
		if o == def {
			marker = "* "
		}
		fmt.Fprintf(out, "  %s%d) %s\n", marker, i+1, o)
	}
	if def != "" {
		fmt.Fprintf(out, "Choose [%s]: ", def)
	} else {
		fmt.Fprintf(out, "Choose: ")
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(options) {
		return options[n-1]
	}
	if contains(options, line) {
		return line
	}
	return def
}

// askLine prompts for free-text, returning the trimmed input or def on blank.
func askLine(r *bufio.Reader, out io.Writer, label, def string) string {
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// stringOptions builds huh select options whose label and value are identical.
func stringOptions(values []string) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(values))
	for _, v := range values {
		opts = append(opts, huh.NewOption(v, v))
	}
	return opts
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// inputIsTerminal reports whether in is an interactive terminal, used to pick
// the rich select UI versus the line-based fallback.
func inputIsTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	return xterm.IsTerminal(f.Fd())
}

// redact masks a secret for display, preserving only a short prefix hint.
func redact(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}
