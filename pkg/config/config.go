package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hmans/beans/pkg/bean"
	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the name of the config file at project root
	ConfigFileName = ".beans.yml"
	// DefaultBeansPath is the default directory for storing beans
	DefaultBeansPath = ".beans"
	// LegacyConfigFile is the old config file location (deprecated)
	LegacyConfigFile = "config.yaml"
	// DefaultServerPort is the default port for the web server
	DefaultServerPort = 8080
	// AnchorRepoRoot anchors the beans directory at the main worktree's root
	// instead of at the config file's directory. Secondary worktrees then share
	// the main workdir's store rather than each carrying their own.
	AnchorRepoRoot = "repo-root"
	// DefaultCommitField is the front matter key used to record a commit SHA
	// when beans.commit_field is not configured.
	DefaultCommitField = "commit"
	// LeafRank is the rank of a type that carries no children. A type without
	// an explicit rank lands here, which keeps the old behaviour where an
	// unknown type was treated like "task".
	LeafRank = 4
)

// DefaultStatuses defines the built-in status table. Config.Statuses can
// override individual entries; see (*Config).StatusList.
// Order determines sort priority: in-progress first (active work), then todo, draft, and done states last.
var DefaultStatuses = []StatusConfig{
	{Name: "in-progress", Color: "peach", Description: "Currently being worked on"},
	{Name: "todo", Color: "green", Description: "Ready to be worked on"},
	{Name: "draft", Color: "overlay2", Description: "Needs refinement before it can be worked on"},
	{Name: "completed", Color: "overlay1", Archive: true, Description: "Finished successfully"},
	{Name: "scrapped", Color: "surface2", Archive: true, Description: "Will not be done"},
}

// DefaultTypes defines the built-in type table. Config.Types can override
// individual entries; see (*Config).TypeList.
var DefaultTypes = []TypeConfig{
	{Name: "milestone", Color: "mauve", Rank: 1, Short: "M", Emphasis: true, Description: "A target release or checkpoint; group work that should ship together"},
	{Name: "epic", Color: "blue", Rank: 2, Short: "E", Emphasis: true, Description: "A thematic container for related work; should have child beans, not be worked on directly"},
	{Name: "feature", Color: "sapphire", Rank: 3, Short: "F", Description: "A user-facing capability or enhancement"},
	{Name: "bug", Color: "maroon", Rank: LeafRank, Short: "B", Description: "Something that is broken and needs fixing"},
	{Name: "task", Color: "", Rank: LeafRank, Short: "T", Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
}

// DefaultPriorities defines the built-in priority table. Config.Priorities
// can override individual entries; see (*Config).PriorityList.
// Priorities are ordered from highest to lowest urgency.
var DefaultPriorities = []PriorityConfig{
	{Name: "critical", Color: "red", Description: "Urgent, blocking work. When possible, address immediately"},
	{Name: "high", Color: "yellow", Description: "Important, should be done before normal work"},
	{Name: "normal", Color: "", Description: "Standard priority"},
	{Name: "low", Color: "overlay0", Description: "Less important, can be delayed"},
	{Name: "deferred", Color: "overlay0", Description: "Explicitly pushed back, avoid doing unless necessary"},
}

// StatusConfig defines a single status with its display color.
type StatusConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Archive     bool   `yaml:"archive,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// TypeConfig defines a single bean type with its display color.
type TypeConfig struct {
	Name  string `yaml:"name"`
	Color string `yaml:"color"`
	// Rank carries the hierarchy: a parent is valid when the child's rank is
	// strictly greater. Ranks 1 to 3 are container ranks, 4 is the leaf rank.
	// Ranks may be left unoccupied; the rule tolerates the gap.
	Rank int `yaml:"rank,omitempty"`
	// Emphasis renders this type bold across type word, id and title. It is
	// what carries the hierarchy where hue alone cannot: Catppuccin tones are
	// uniformly pastel, so a container would otherwise lose against a leaf.
	Emphasis bool `yaml:"emphasis,omitempty"`
	// Roadmap decides whether this type gets its own section in `beans roadmap`
	// and an entry in `beans milestones`. Nil means visible; only an explicit
	// roadmap: false hides the type, and it hides its whole subtree with it.
	Roadmap *bool `yaml:"roadmap,omitempty"`
	// Short is the single-character code the narrow list view renders. Empty
	// means the first letter of the name, upper-cased.
	Short       string `yaml:"short,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// PriorityConfig defines a single priority level with its display color.
type PriorityConfig struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// StatusOverride is one entry of Config.Statuses: a status the config wants
// to override (matched by Name) or add. Archive is a pointer so an omitted
// key is structurally distinct from an explicit "archive: false" - a plain
// bool cannot make that distinction, and collapsing it silently flips a
// status like "completed" back to non-archiving on any recolour-only edit.
type StatusOverride struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color,omitempty"`
	Description string `yaml:"description,omitempty"`
	Archive     *bool  `yaml:"archive,omitempty"`
}

// TypeOverride is one entry of Config.Types: a type the config wants to
// override (matched by Name) or add. Emphasis is a pointer for the same
// reason StatusOverride.Archive is: an omitted key must stay structurally
// distinct from an explicit "emphasis: false", or a colour-only recolour of
// e.g. "milestone" would silently clear its emphasis.
type TypeOverride struct {
	Name  string `yaml:"name"`
	Color string `yaml:"color,omitempty"`
	// Rank is a pointer for the same reason Emphasis is: an omitted key must
	// stay distinct from an explicit rank, or a colour-only override would
	// pull its type down to rank 0 and outrank every container.
	Rank        *int   `yaml:"rank,omitempty"`
	Description string `yaml:"description,omitempty"`
	Emphasis    *bool  `yaml:"emphasis,omitempty"`
	// Roadmap decides whether this type gets its own section in `beans roadmap`
	// and an entry in `beans milestones`. Nil means visible; only an explicit
	// roadmap: false hides the type, and it hides its whole subtree with it.
	Roadmap *bool `yaml:"roadmap,omitempty"`
	// Short is the single-character code the narrow list view renders. Empty
	// means the first letter of the name, upper-cased.
	Short string `yaml:"short,omitempty"`
}

// PriorityOverride is one entry of Config.Priorities: a priority the config
// wants to override (matched by Name) or add.
type PriorityOverride struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// PermissionMode represents the default agent permission mode.
type PermissionMode string

const (
	PermissionModeAct  PermissionMode = "act"
	PermissionModePlan PermissionMode = "plan"
)

// IntegrateMode represents the worktree integration strategy.
type IntegrateMode string

const (
	IntegrateModeLocal IntegrateMode = "local"
	IntegrateModePR    IntegrateMode = "pr"
)

// WorktreeConfig defines settings for git worktree management.
type WorktreeConfig struct {
	// BaseRef is the git ref to use as the starting point for new worktree branches.
	// Default: "main"
	BaseRef string `yaml:"base_ref,omitempty"`

	// Path is the directory where worktrees are created.
	// Default: ~/.beans/worktrees/<project-name>/
	// Supports ~ for home directory.
	Path string `yaml:"path,omitempty"`

	// Setup is a shell command to run inside a worktree after creation (e.g. "pnpm install").
	Setup string `yaml:"setup,omitempty"`

	// Run is a shell command to run the project (e.g. "mise dev").
	// When set, a "Run" button appears in the workspace toolbar.
	Run string `yaml:"run,omitempty"`

	// Integrate controls the worktree integration strategy.
	// "local" (default): squash-merge locally, hides PR buttons.
	// "pr": push and create PRs, hides the local Integrate button.
	Integrate IntegrateMode `yaml:"integrate,omitempty"`

	// FetchTimeout is the timeout in seconds for the git fetch that runs before
	// creating a new worktree. This fetch updates the base ref from the remote.
	// Set to 0 to disable the fetch entirely (useful for airgapped environments).
	// Default: 10 (seconds).
	FetchTimeout *int `yaml:"fetch_timeout,omitempty"`
}

// AgentConfig defines settings for agent sessions.
type AgentConfig struct {
	// Enabled controls whether agent functionality is available.
	// When false, the web UI hides agent chats, status panes, and worktree features.
	// Default: true
	Enabled *bool `yaml:"enabled,omitempty"`

	// DefaultMode is the default mode for new agent sessions.
	// Valid values: "act" (fully autonomous), "plan" (read-only).
	// Default: "act"
	DefaultMode PermissionMode `yaml:"default_mode,omitempty"`

	// DefaultEffort is the default thinking effort level for new agent sessions.
	// Valid values: "low", "medium", "high", "max".
	// When omitted, new sessions start with no effort override (uses CLI default).
	DefaultEffort string `yaml:"default_effort,omitempty"`
}

// ProjectConfig defines project-level settings.
type ProjectConfig struct {
	// Name is the human-readable project name, displayed in the UI.
	// Default: derived from directory name during `beans init`.
	Name string `yaml:"name,omitempty"`
}

// ServerConfig defines settings for the web server.
type ServerConfig struct {
	// Port is the port to listen on (default: 8080)
	Port int `yaml:"port,omitempty"`
	// CORSOrigins is the list of allowed origins for CORS and WebSocket.
	// Supports exact origins and port wildcards (e.g. "http://localhost:*").
	// Use "*" to allow all origins (not recommended for production).
	// Default: ["http://localhost:*", "http://127.0.0.1:*"]
	CORSOrigins []string `yaml:"cors_origins,omitempty"`
}

// DisplayConfig controls how the CLI renders to a terminal.
type DisplayConfig struct {
	// Theme names one of the bundled Catppuccin flavours: latte, frappe,
	// macchiato or mocha. Empty means mocha.
	Theme string `yaml:"theme,omitempty"`

	// MaxWidth caps the rendered width. 0 means unset and yields 110;
	// -1 disables the cap entirely.
	MaxWidth int `yaml:"max_width,omitempty"`
}

// Config holds the beans configuration.
type Config struct {
	Project  ProjectConfig  `yaml:"project,omitempty"`
	Beans    BeansConfig    `yaml:"beans"`
	Worktree WorktreeConfig `yaml:"worktree,omitempty"`
	Agent    AgentConfig    `yaml:"agent,omitempty"`
	Server   ServerConfig   `yaml:"server,omitempty"`
	Display  DisplayConfig  `yaml:"display,omitempty"`

	// Statuses, Types and Priorities override the built-in tables entry by
	// entry: a config naming only "bug" keeps the other four types as they are.
	// A name the defaults do not carry is appended.
	Statuses   []StatusOverride   `yaml:"statuses,omitempty"`
	Types      []TypeOverride     `yaml:"types,omitempty"`
	Priorities []PriorityOverride `yaml:"priorities,omitempty"`

	// configDir is the directory containing the config file (not serialized)
	// Used to resolve relative paths
	configDir string `yaml:"-"`

	// configFile is the .beans.yml this config was read from (not serialized).
	// Empty when no file was found and the defaults are in play; callers use
	// that distinction to tell a repository's own declaration apart from a
	// fallback (see ConfigFile).
	configFile string `yaml:"-"`
}

// BeansConfig defines settings for bean creation.
type BeansConfig struct {
	// Path is the path to the beans directory (relative to config file location)
	Path string `yaml:"path,omitempty"`
	// Anchor decides what Path is relative to. Empty (the default) keeps the
	// config file's own directory, so a secondary worktree gets its own store.
	// AnchorRepoRoot resolves against the main worktree's root instead, so every
	// worktree of a repository shares one store.
	Anchor         string `yaml:"anchor,omitempty"`
	Prefix         string `yaml:"prefix"`
	IDLength       int    `yaml:"id_length"`
	DefaultStatus  string `yaml:"default_status,omitempty"`
	DefaultType    string `yaml:"default_type,omitempty"`
	RequireIfMatch bool   `yaml:"require_if_match,omitempty"`
	// RequireFieldsOn maps a target status to front matter keys that must carry
	// a non-empty value whenever a bean is written into that status.
	RequireFieldsOn map[string][]string `yaml:"require_fields_on,omitempty"`
	// CommitField names the extra front matter key that holds a git commit SHA.
	// Only this key is git-verified when written or checked.
	CommitField string `yaml:"commit_field,omitempty"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Beans: BeansConfig{
			Path:          DefaultBeansPath,
			Prefix:        "",
			IDLength:      4,
			DefaultStatus: "todo",
			DefaultType:   "task",
		},
		Worktree: WorktreeConfig{
			BaseRef:   DefaultWorktreeBaseRef,
			Integrate: IntegrateModeLocal,
		},
		Agent: AgentConfig{
			Enabled:     boolPtr(true),
			DefaultMode: PermissionModeAct,
		},
		Server: ServerConfig{
			Port: DefaultServerPort,
		},
	}
}

// DefaultWithPrefix returns a Config with the given prefix.
func DefaultWithPrefix(prefix string) *Config {
	cfg := Default()
	cfg.Beans.Prefix = prefix
	return cfg
}

// FindConfig searches upward from the given directory for a .beans.yml config file.
// Returns the absolute path to the config file, or empty string if not found.
func FindConfig(startDir string) (string, error) {
	return FindConfigWithin(startDir, "")
}

// FindConfigWithin searches upward from startDir for a .beans.yml config file,
// stopping at rootDir (inclusive). If rootDir is empty, searches up to the
// filesystem root.
func FindConfigWithin(startDir, rootDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	var absRoot string
	if rootDir != "" {
		absRoot, err = filepath.Abs(rootDir)
		if err != nil {
			return "", err
		}
	}

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Stop if we've reached the root boundary
		if absRoot != "" && dir == absRoot {
			return "", nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

// Load reads configuration from the given config file path.
// Returns default config if the file doesn't exist.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Store the config file and its directory: the directory resolves
	// relative paths, the file records that a declaration really exists.
	cfg.configDir = filepath.Dir(configPath)
	cfg.configFile = configPath

	// Apply defaults for missing values
	if cfg.Beans.Path == "" {
		cfg.Beans.Path = DefaultBeansPath
	}
	if cfg.Beans.IDLength == 0 {
		cfg.Beans.IDLength = 4
	}
	if cfg.Beans.DefaultStatus == "" {
		cfg.Beans.DefaultStatus = "todo"
	}
	if cfg.Beans.DefaultType == "" {
		// DefaultTypes[0].Name via the merged list: harmless today (an
		// override never changes an entry's Name, only its Color and
		// Description), kept this way only because a future merge that did
		// allow renaming should not have to remember this call site too.
		cfg.Beans.DefaultType = cfg.TypeList()[0].Name
	}

	// A misspelt anchor must not degrade into the default: that would silently
	// resolve a different store than the file asks for.
	if err := ValidateAnchor(cfg.Beans.Anchor); err != nil {
		return nil, fmt.Errorf("%s: %w", configPath, err)
	}

	if err := cfg.validateRequireFieldsOn(); err != nil {
		return nil, fmt.Errorf("%s: %w", configPath, err)
	}

	return &cfg, nil
}

// ValidateAnchor rejects any beans.anchor value the resolver cannot honour.
func ValidateAnchor(anchor string) error {
	switch anchor {
	case "", AnchorRepoRoot:
		return nil
	default:
		return fmt.Errorf("unknown beans.anchor %q (expected %q or an empty value)", anchor, AnchorRepoRoot)
	}
}

// validateRequireFieldsOn rejects a beans.require_fields_on configuration that
// names an unknown status, an empty field name, or a field beans already
// manages as a schema field. Misconfiguration must not silently degrade into
// no policy at all.
func (c *Config) validateRequireFieldsOn() error {
	for status, fields := range c.Beans.RequireFieldsOn {
		if !c.IsValidStatus(status) {
			return fmt.Errorf("unknown status %q in beans.require_fields_on (valid: %s)", status, strings.Join(c.StatusNames(), ", "))
		}
		for _, field := range fields {
			if strings.TrimSpace(field) == "" {
				return fmt.Errorf("beans.require_fields_on[%q] contains an empty field name", status)
			}
			if _, known := bean.ReservedKeyFlag(field); known {
				return fmt.Errorf("beans.require_fields_on[%q] names reserved front matter field %q (managed by beans, not an extra key)", status, field)
			}
		}
	}
	return nil
}

// LoadFromDirectory finds and loads the config file by searching upward from the given directory.
// If no config file is found, returns a default config anchored at the given directory.
func LoadFromDirectory(startDir string) (*Config, error) {
	return LoadFromDirectoryWithin(startDir, "")
}

// LoadFromDirectoryWithin is like LoadFromDirectory but limits the search to
// within rootDir (see FindConfigWithin).
func LoadFromDirectoryWithin(startDir, rootDir string) (*Config, error) {
	configPath, err := FindConfigWithin(startDir, rootDir)
	if err != nil {
		return nil, err
	}

	if configPath == "" {
		// No config found, return default anchored at startDir
		cfg := Default()
		cfg.configDir = startDir
		return cfg, nil
	}

	return Load(configPath)
}

// ResolveBeansPath returns the absolute path to the beans directory.
func (c *Config) ResolveBeansPath() string {
	if filepath.IsAbs(c.Beans.Path) {
		return c.Beans.Path
	}
	if c.configDir == "" {
		// Fallback: use current directory
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, c.Beans.Path)
	}
	return filepath.Join(c.configDir, c.Beans.Path)
}

// ConfigDir returns the directory containing the config file.
func (c *Config) ConfigDir() string {
	return c.configDir
}

// ConfigFile returns the path of the .beans.yml this config was read from, or
// an empty string when no config file was found and the defaults apply. A
// non-empty result means the repository declared where its store lives, which
// ranks above the BEANS_PATH env var during path resolution.
func (c *Config) ConfigFile() string {
	return c.configFile
}

// SetConfigDir sets the config directory (for testing or when creating new configs).
func (c *Config) SetConfigDir(dir string) {
	c.configDir = dir
}

// Save writes the configuration to the config file atomically with helpful comments.
// Writes to a temporary file in the same directory, fsyncs, and renames to the target.
// Preserves the existing file mode if the file already exists.
// If configDir is set, saves to that directory; otherwise saves to the given directory.
func (c *Config) Save(dir string) error {
	targetDir := c.configDir
	if targetDir == "" {
		targetDir = dir
	}
	path := filepath.Join(targetDir, ConfigFileName)

	doc := c.toYAMLNode()

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("closing encoder: %w", err)
	}

	// Get the existing file mode if the target file exists
	fileMode := os.FileMode(0644) // default mode
	if stat, err := os.Stat(path); err == nil {
		fileMode = stat.Mode()
	}

	// Write to a temporary file in the same directory to ensure atomicity.
	tmpFile, err := os.CreateTemp(targetDir, ConfigFileName+".tmp.*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file if it still exists

	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	// Fsync to ensure the data hits disk before we rename
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomic rename from temp to target
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming config file: %w", err)
	}

	// Restore the original file mode if it was different
	if err := os.Chmod(path, fileMode); err != nil {
		return fmt.Errorf("restoring file mode: %w", err)
	}

	return nil
}

// toYAMLNode builds a yaml.Node document tree with inline comments.
func (c *Config) toYAMLNode() *yaml.Node {
	// Helper to create a scalar node
	scalar := func(value string, tag string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: value, Tag: tag}
	}
	strNode := func(value string) *yaml.Node {
		return scalar(value, "!!str")
	}
	intNode := func(value int) *yaml.Node {
		return scalar(fmt.Sprintf("%d", value), "!!int")
	}

	// Build the project mapping
	projectMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Project.Name != "" {
		key := strNode("name")
		key.HeadComment = "Human-readable project name (displayed in the UI)"
		projectMapping.Content = append(projectMapping.Content, key, strNode(c.Project.Name))
	}

	// Build the beans mapping
	beansMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	if c.Beans.Path != "" {
		key := strNode("path")
		key.HeadComment = "Directory where bean files are stored"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.Path))
	}

	if c.Beans.Anchor != "" {
		key := strNode("anchor")
		key.HeadComment = "What `path` is relative to: \"repo-root\" makes every worktree share the main workdir's store"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.Anchor))
	}

	prefixKey := strNode("prefix")
	prefixKey.HeadComment = "Prefix for bean IDs (e.g., \"myproject-abc1\")"
	beansMapping.Content = append(beansMapping.Content, prefixKey, strNode(c.Beans.Prefix))

	idLenKey := strNode("id_length")
	idLenKey.HeadComment = "Length of the random ID suffix"
	beansMapping.Content = append(beansMapping.Content, idLenKey, intNode(c.Beans.IDLength))

	if c.Beans.DefaultStatus != "" {
		key := strNode("default_status")
		key.HeadComment = "Default status for new beans"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.DefaultStatus))
	}

	if c.Beans.DefaultType != "" {
		key := strNode("default_type")
		key.HeadComment = "Default type for new beans"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.DefaultType))
	}

	if c.Beans.RequireIfMatch {
		key := strNode("require_if_match")
		key.HeadComment = "Require ETag for updates (optimistic concurrency)"
		beansMapping.Content = append(beansMapping.Content, key, scalar("true", "!!bool"))
	}

	if len(c.Beans.RequireFieldsOn) > 0 {
		key := strNode("require_fields_on")
		key.HeadComment = "Front matter keys that must be set when a bean enters a status"
		mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, s := range c.StatusList() { // deterministic key order
			fields := c.Beans.RequireFieldsOn[s.Name]
			if len(fields) == 0 {
				continue
			}
			seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
			for _, f := range fields {
				seq.Content = append(seq.Content, strNode(f))
			}
			mapping.Content = append(mapping.Content, strNode(s.Name), seq)
		}
		beansMapping.Content = append(beansMapping.Content, key, mapping)
	}

	if c.Beans.CommitField != "" {
		key := strNode("commit_field")
		key.HeadComment = "Extra front matter key holding the git commit SHA (default: commit)"
		beansMapping.Content = append(beansMapping.Content, key, strNode(c.Beans.CommitField))
	}

	// Build the worktree mapping
	worktreeMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Worktree.BaseRef != "" {
		key := strNode("base_ref")
		key.HeadComment = "Git ref to use as the base for new worktree branches (default: main)"
		worktreeMapping.Content = append(worktreeMapping.Content, key, strNode(c.Worktree.BaseRef))
	}
	if c.Worktree.Path != "" {
		key := strNode("path")
		key.HeadComment = "Directory for worktrees (default: ~/.beans/worktrees/<project>/)"
		worktreeMapping.Content = append(worktreeMapping.Content, key, strNode(c.Worktree.Path))
	}
	setupKey := strNode("setup")
	setupKey.HeadComment = "Shell command to run inside a worktree after creation (e.g. \"pnpm install\")"
	worktreeMapping.Content = append(worktreeMapping.Content, setupKey, strNode(c.Worktree.Setup))

	runKey := strNode("run")
	runKey.HeadComment = "Shell command to run the project (adds a \"Run\" button to workspace toolbar)"
	worktreeMapping.Content = append(worktreeMapping.Content, runKey, strNode(c.Worktree.Run))

	integrateKey := strNode("integrate")
	integrateKey.HeadComment = "Integration strategy: \"local\" (squash-merge locally) or \"pr\" (push and create PRs)"
	worktreeMapping.Content = append(worktreeMapping.Content, integrateKey, strNode(string(c.GetWorktreeIntegrate())))

	// Build the agent mapping
	agentMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Agent.Enabled != nil {
		key := strNode("enabled")
		key.HeadComment = "Enable agent functionality in the web UI (true, false)"
		agentMapping.Content = append(agentMapping.Content, key, scalar(fmt.Sprintf("%t", *c.Agent.Enabled), "!!bool"))
	}
	if c.Agent.DefaultMode != "" {
		key := strNode("default_mode")
		key.HeadComment = "Default mode for agent sessions (act, plan)"
		agentMapping.Content = append(agentMapping.Content, key, strNode(string(c.Agent.DefaultMode)))
	}
	// Build the server mapping
	serverMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Server.Port != 0 {
		portKey := strNode("port")
		portKey.HeadComment = "Port for the web UI (used by `beans-serve`)"
		serverMapping.Content = append(serverMapping.Content, portKey, intNode(c.Server.Port))
	}

	// Build the display mapping. Like Statuses/Types/Priorities, this must be
	// round-tripped: Save() is called by `beans rename` on an already-loaded
	// Config, and before this a configured theme/max_width was silently
	// dropped on the next write.
	displayMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if c.Display.Theme != "" {
		key := strNode("theme")
		key.HeadComment = "Catppuccin flavour: latte, frappe, macchiato or mocha (default: mocha)"
		displayMapping.Content = append(displayMapping.Content, key, strNode(c.Display.Theme))
	}
	if c.Display.MaxWidth != 0 {
		key := strNode("max_width")
		key.HeadComment = "Caps the rendered width (default: 110); -1 disables the cap"
		displayMapping.Content = append(displayMapping.Content, key, intNode(c.Display.MaxWidth))
	}

	// Build the statuses/types/priorities override sequences. These round-trip
	// exactly what was configured (not the merged result) so that Save() -
	// called by `beans rename`, for instance - never silently drops a user's
	// overrides from the file.
	statusesSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, s := range c.Statuses {
		m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		m.Content = append(m.Content, strNode("name"), strNode(s.Name))
		if s.Color != "" {
			m.Content = append(m.Content, strNode("color"), strNode(s.Color))
		}
		if s.Archive != nil {
			m.Content = append(m.Content, strNode("archive"), scalar(fmt.Sprintf("%t", *s.Archive), "!!bool"))
		}
		if s.Description != "" {
			m.Content = append(m.Content, strNode("description"), strNode(s.Description))
		}
		statusesSeq.Content = append(statusesSeq.Content, m)
	}

	typesSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, t := range c.Types {
		m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		m.Content = append(m.Content, strNode("name"), strNode(t.Name))
		if t.Color != "" {
			m.Content = append(m.Content, strNode("color"), strNode(t.Color))
		}
		if t.Rank != nil {
			m.Content = append(m.Content, strNode("rank"), scalar(fmt.Sprintf("%d", *t.Rank), "!!int"))
		}
		if t.Emphasis != nil {
			m.Content = append(m.Content, strNode("emphasis"), scalar(fmt.Sprintf("%t", *t.Emphasis), "!!bool"))
		}
		if t.Roadmap != nil {
			m.Content = append(m.Content, strNode("roadmap"), scalar(fmt.Sprintf("%t", *t.Roadmap), "!!bool"))
		}
		if t.Short != "" {
			m.Content = append(m.Content, strNode("short"), strNode(t.Short))
		}
		if t.Description != "" {
			m.Content = append(m.Content, strNode("description"), strNode(t.Description))
		}
		typesSeq.Content = append(typesSeq.Content, m)
	}

	prioritiesSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, p := range c.Priorities {
		m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		m.Content = append(m.Content, strNode("name"), strNode(p.Name))
		if p.Color != "" {
			m.Content = append(m.Content, strNode("color"), strNode(p.Color))
		}
		if p.Description != "" {
			m.Content = append(m.Content, strNode("description"), strNode(p.Description))
		}
		prioritiesSeq.Content = append(prioritiesSeq.Content, m)
	}

	// Build the top-level mapping
	topMapping := &yaml.Node{
		Kind:        yaml.MappingNode,
		Tag:         "!!map",
		HeadComment: "Beans configuration\nSee: https://github.com/hmans/beans",
	}
	if len(projectMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("project"), projectMapping)
	}

	topMapping.Content = append(topMapping.Content, strNode("beans"), beansMapping)

	if len(worktreeMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("worktree"), worktreeMapping)
	}

	if len(agentMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("agent"), agentMapping)
	}

	if len(serverMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("server"), serverMapping)
	}

	if len(displayMapping.Content) > 0 {
		topMapping.Content = append(topMapping.Content, strNode("display"), displayMapping)
	}

	if len(statusesSeq.Content) > 0 {
		key := strNode("statuses")
		key.HeadComment = "Status overrides, merged entry by entry with the built-in defaults"
		topMapping.Content = append(topMapping.Content, key, statusesSeq)
	}

	if len(typesSeq.Content) > 0 {
		key := strNode("types")
		key.HeadComment = "Type overrides, merged entry by entry with the built-in defaults"
		topMapping.Content = append(topMapping.Content, key, typesSeq)
	}

	if len(prioritiesSeq.Content) > 0 {
		key := strNode("priorities")
		key.HeadComment = "Priority overrides, merged entry by entry with the built-in defaults"
		topMapping.Content = append(topMapping.Content, key, prioritiesSeq)
	}

	// Wrap in a document node
	return &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{topMapping},
	}
}

// IsValidStatus returns true if the status is a known status name, built-in
// or configured.
func (c *Config) IsValidStatus(status string) bool {
	for _, s := range c.StatusList() {
		if s.Name == status {
			return true
		}
	}
	return false
}

// RequiredFieldsFor returns the front matter keys that must be set when a bean
// enters status. Returns nil when no policy applies.
func (c *Config) RequiredFieldsFor(status string) []string {
	return c.Beans.RequireFieldsOn[status]
}

// GetCommitField returns the configured commit field name, or DefaultCommitField.
func (c *Config) GetCommitField() string {
	if c.Beans.CommitField != "" {
		return c.Beans.CommitField
	}
	return DefaultCommitField
}

// mergeDefaults merges a config's override entries into a copy of a
// built-in default table, matched by name. An override naming an entry the
// defaults already have is applied onto that entry via apply; an override
// naming an unknown entry is appended as a new one (apply runs against a
// zero-value T, so it must itself set every field the new entry needs,
// starting with the name). This is the single place the three list merges
// (statuses, types, priorities) share, so a merge rule such as "only an
// explicit field touches the default" is written once, not duplicated per
// type - see StatusList's Archive handling for why that duplication was the
// actual defect the previous copy-pasted version hid.
func mergeDefaults[T, O any](defaults []T, overrides []O, key func(T) string, name func(O) string, apply func(*T, O)) []T {
	out := make([]T, len(defaults))
	copy(out, defaults)
	for _, override := range overrides {
		merged := false
		for i := range out {
			if key(out[i]) != name(override) {
				continue
			}
			apply(&out[i], override)
			merged = true
			break
		}
		if !merged {
			var fresh T
			apply(&fresh, override)
			out = append(out, fresh)
		}
	}
	return out
}

// StatusList returns the built-in statuses with any configured overrides
// applied. A config entry overrides its named status field by field; a name
// the defaults do not carry is appended, so custom statuses are possible
// without this being their dedicated feature.
func (c *Config) StatusList() []StatusConfig {
	if c == nil {
		out := make([]StatusConfig, len(DefaultStatuses))
		copy(out, DefaultStatuses)
		return out
	}
	return mergeDefaults(DefaultStatuses, c.Statuses,
		func(t StatusConfig) string { return t.Name },
		func(o StatusOverride) string { return o.Name },
		func(t *StatusConfig, o StatusOverride) {
			t.Name = o.Name
			if o.Color != "" {
				t.Color = o.Color
			}
			if o.Description != "" {
				t.Description = o.Description
			}
			// Archive is a pointer: only an explicit archive: key (true or
			// false) touches it, so a colour-only override cannot flip a
			// status like "completed" back to non-archiving by accident.
			if o.Archive != nil {
				t.Archive = *o.Archive
			}
		},
	)
}

// StatusNames returns the names of the merged status list (see StatusList).
func (c *Config) StatusNames() []string {
	list := c.StatusList()
	names := make([]string, len(list))
	for i, s := range list {
		names[i] = s.Name
	}
	return names
}

// GetStatus returns the StatusConfig for a status name from the merged list
// (see StatusList), or nil if unknown.
func (c *Config) GetStatus(name string) *StatusConfig {
	list := c.StatusList()
	for i := range list {
		if list[i].Name == name {
			return &list[i]
		}
	}
	return nil
}

// GetDefaultStatus returns the default status name for new beans.
func (c *Config) GetDefaultStatus() string {
	if c.Beans.DefaultStatus == "" {
		return "todo"
	}
	return c.Beans.DefaultStatus
}

// GetDefaultType returns the default type name for new beans.
func (c *Config) GetDefaultType() string {
	return c.Beans.DefaultType
}

// IsArchiveStatus returns true if the given status is marked for archiving.
func (c *Config) IsArchiveStatus(name string) bool {
	if s := c.GetStatus(name); s != nil {
		return s.Archive
	}
	return false
}

// GetType returns the TypeConfig for a type name from the merged list (see
// TypeList), or nil if unknown.
func (c *Config) GetType(name string) *TypeConfig {
	list := c.TypeList()
	for i := range list {
		if list[i].Name == name {
			return &list[i]
		}
	}
	return nil
}

// RankOf returns the hierarchy rank of a type name. An unknown type and a
// type without an explicit rank both sit at LeafRank.
func (c *Config) RankOf(typeName string) int {
	if t := c.GetType(typeName); t != nil && t.Rank != 0 {
		return t.Rank
	}
	return LeafRank
}

// ShortOf returns the single-character code for a type: the configured one,
// otherwise the upper-cased first letter, and "?" for an unknown type.
func (c *Config) ShortOf(typeName string) string {
	t := c.GetType(typeName)
	if t == nil {
		return "?"
	}
	if t.Short != "" {
		return t.Short
	}
	if t.Name == "" {
		return "?"
	}
	return strings.ToUpper(t.Name[:1])
}

// IsRoadmapType reports whether a type appears as its own container in the
// aggregate views. Anything not explicitly switched off is visible.
func (c *Config) IsRoadmapType(typeName string) bool {
	if t := c.GetType(typeName); t != nil && t.Roadmap != nil {
		return *t.Roadmap
	}
	return true
}

// TypesAtRank returns the names of every type on the given rank, in the order
// the merged list carries them.
func (c *Config) TypesAtRank(rank int) []string {
	var names []string
	for _, t := range c.TypeList() {
		r := t.Rank
		if r == 0 {
			r = LeafRank
		}
		if r == rank {
			names = append(names, t.Name)
		}
	}
	return names
}

// TypeNames returns the names of the merged type list (see TypeList).
func (c *Config) TypeNames() []string {
	list := c.TypeList()
	names := make([]string, len(list))
	for i, t := range list {
		names[i] = t.Name
	}
	return names
}

// IsValidType returns true if the type is a known type name, built-in or
// configured.
func (c *Config) IsValidType(typeName string) bool {
	for _, t := range c.TypeList() {
		if t.Name == typeName {
			return true
		}
	}
	return false
}

// TypeList returns the built-in types with any configured overrides applied.
// A config entry overrides its named type field by field; a name the
// defaults do not carry is appended.
func (c *Config) TypeList() []TypeConfig {
	if c == nil {
		out := make([]TypeConfig, len(DefaultTypes))
		copy(out, DefaultTypes)
		return out
	}
	return mergeDefaults(DefaultTypes, c.Types,
		func(t TypeConfig) string { return t.Name },
		func(o TypeOverride) string { return o.Name },
		func(t *TypeConfig, o TypeOverride) {
			t.Name = o.Name
			if o.Color != "" {
				t.Color = o.Color
			}
			if o.Description != "" {
				t.Description = o.Description
			}
			if o.Rank != nil {
				t.Rank = *o.Rank
			}
			// Emphasis is a pointer: only an explicit emphasis: key (true or
			// false) touches it, so a colour-only override cannot flip a
			// type like "milestone" back to non-emphasised by accident.
			if o.Emphasis != nil {
				t.Emphasis = *o.Emphasis
			}
			// Roadmap is a pointer for the same reason: an omitted key must
			// stay distinct from an explicit "roadmap: false", or a
			// colour-only override would silently hide the type.
			if o.Roadmap != nil {
				t.Roadmap = o.Roadmap
			}
			if o.Short != "" {
				t.Short = o.Short
			}
			// An appended type starts from a zero TypeConfig, so an entry
			// without rank: would sit at rank 0 and outrank every container.
			// Leaf rank is the safe default and matches the old behaviour for
			// unknown types.
			if t.Rank == 0 {
				t.Rank = LeafRank
			}
		},
	)
}

// BeanColors holds resolved color information for rendering a bean
type BeanColors struct {
	StatusColor   string
	TypeColor     string
	PriorityColor string
	IsArchive     bool
}

// GetBeanColors returns the resolved colors for a bean based on its status, type, and priority.
func (c *Config) GetBeanColors(status, typeName, priority string) BeanColors {
	colors := BeanColors{
		StatusColor:   "gray",
		TypeColor:     "",
		PriorityColor: "",
		IsArchive:     false,
	}

	if statusCfg := c.GetStatus(status); statusCfg != nil {
		colors.StatusColor = statusCfg.Color
	}
	colors.IsArchive = c.IsArchiveStatus(status)

	if typeCfg := c.GetType(typeName); typeCfg != nil {
		colors.TypeColor = typeCfg.Color
	}

	if priorityCfg := c.GetPriority(priority); priorityCfg != nil {
		colors.PriorityColor = priorityCfg.Color
	}

	return colors
}

// GetPriority returns the PriorityConfig for a priority name from the merged
// list (see PriorityList), or nil if unknown.
func (c *Config) GetPriority(name string) *PriorityConfig {
	list := c.PriorityList()
	for i := range list {
		if list[i].Name == name {
			return &list[i]
		}
	}
	return nil
}

// PriorityNames returns the names of the merged priority list (see
// PriorityList), in order from highest to lowest.
func (c *Config) PriorityNames() []string {
	list := c.PriorityList()
	names := make([]string, len(list))
	for i, p := range list {
		names[i] = p.Name
	}
	return names
}

// IsValidPriority returns true if the priority is a known priority name,
// built-in or configured. Empty string is valid (means no priority set).
func (c *Config) IsValidPriority(priority string) bool {
	if priority == "" {
		return true
	}
	for _, p := range c.PriorityList() {
		if p.Name == priority {
			return true
		}
	}
	return false
}

// PriorityList returns the built-in priorities with any configured overrides
// applied. A config entry overrides its named priority field by field; a
// name the defaults do not carry is appended.
func (c *Config) PriorityList() []PriorityConfig {
	if c == nil {
		out := make([]PriorityConfig, len(DefaultPriorities))
		copy(out, DefaultPriorities)
		return out
	}
	return mergeDefaults(DefaultPriorities, c.Priorities,
		func(t PriorityConfig) string { return t.Name },
		func(o PriorityOverride) string { return o.Name },
		func(t *PriorityConfig, o PriorityOverride) {
			t.Name = o.Name
			if o.Color != "" {
				t.Color = o.Color
			}
			if o.Description != "" {
				t.Description = o.Description
			}
		},
	)
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool { return &b }

// DefaultWorktreeBaseRef is the default base ref for new worktree branches.
const DefaultWorktreeBaseRef = "main"

// ResolveWorktreePath returns the absolute path to the directory where worktrees
// should be created. If worktree.path is configured, it is used (with ~ expansion).
// Otherwise, defaults to ~/.beans/worktrees/<projectName>/.
// projectName is used only when computing the default path.
func (c *Config) ResolveWorktreePath(projectName string) (string, error) {
	if c.Worktree.Path != "" {
		return expandHome(c.Worktree.Path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if projectName == "" {
		projectName = "default"
	}
	return filepath.Join(home, ".beans", "worktrees", projectName), nil
}

// expandHome expands a leading ~ in a path to the user's home directory.
func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	// Relative paths resolve against cwd
	return filepath.Abs(path)
}

// GetWorktreeBaseRef returns the configured base ref for new worktree branches.
// Returns "main" if not set.
func (c *Config) GetWorktreeBaseRef() string {
	if c.Worktree.BaseRef == "" {
		return DefaultWorktreeBaseRef
	}
	return c.Worktree.BaseRef
}

// GetWorktreeSetup returns the configured setup command for new worktrees.
func (c *Config) GetWorktreeSetup() string {
	return c.Worktree.Setup
}

// GetWorktreeRun returns the configured run command for worktrees.
func (c *Config) GetWorktreeRun() string {
	return c.Worktree.Run
}

// GetWorktreeFetchTimeout returns the configured fetch timeout as a time.Duration.
// Returns 10s by default. Returns 0 if explicitly set to 0 (disables fetch).
func (c *Config) GetWorktreeFetchTimeout() time.Duration {
	if c.Worktree.FetchTimeout == nil {
		return 10 * time.Second
	}
	return time.Duration(*c.Worktree.FetchTimeout) * time.Second
}

// GetWorktreeIntegrate returns the configured integration mode.
// Returns "local" if not set or invalid.
func (c *Config) GetWorktreeIntegrate() IntegrateMode {
	switch c.Worktree.Integrate {
	case IntegrateModeLocal, IntegrateModePR:
		return c.Worktree.Integrate
	default:
		return IntegrateModeLocal
	}
}

// IsAgentEnabled returns whether agent functionality is enabled.
// Returns true if not explicitly set.
func (c *Config) IsAgentEnabled() bool {
	if c.Agent.Enabled == nil {
		return true
	}
	return *c.Agent.Enabled
}

// GetDefaultMode returns the configured default permission mode for agent sessions.
// Returns "act" if not set or invalid. Also accepts "yolo" as a backwards-compatible alias.
func (c *Config) GetDefaultMode() PermissionMode {
	switch c.Agent.DefaultMode {
	case PermissionModeAct, PermissionModePlan:
		return c.Agent.DefaultMode
	case "yolo":
		return PermissionModeAct // backwards-compatible alias
	default:
		return PermissionModeAct
	}
}

// GetDefaultEffort returns the raw configured default effort level for agent sessions.
// Returns empty string if not set. Use IsValidEffortLevel to validate before use.
func (c *Config) GetDefaultEffort() string {
	return c.Agent.DefaultEffort
}

// IsValidEffortLevel returns true if the effort level is a valid value.
func IsValidEffortLevel(effort string) bool {
	switch effort {
	case "low", "medium", "high", "max":
		return true
	default:
		return false
	}
}

// IsValidPermissionMode returns true if the mode is a valid permission mode.
func IsValidPermissionMode(mode string) bool {
	switch PermissionMode(mode) {
	case PermissionModeAct, PermissionModePlan, "yolo":
		return true
	default:
		return false
	}
}

// GetProjectName returns the configured project name, or empty string if not set.
func (c *Config) GetProjectName() string {
	return c.Project.Name
}

// GetServerPort returns the configured server port, or the default if not set.
func (c *Config) GetServerPort() int {
	if c.Server.Port == 0 {
		return DefaultServerPort
	}
	return c.Server.Port
}

// GetCORSOrigins returns the configured CORS origins, or the defaults if not set.
func (c *Config) GetCORSOrigins() []string {
	if len(c.Server.CORSOrigins) > 0 {
		return c.Server.CORSOrigins
	}
	return []string{"http://localhost:*", "http://127.0.0.1:*"}
}

// GetTheme returns the configured theme name, or "mocha" when unset.
func (c *Config) GetTheme() string {
	if c.Display.Theme == "" {
		return "mocha"
	}
	return c.Display.Theme
}

// GetMaxWidth returns the configured width cap: 110 when unset, -1 when the
// cap is explicitly disabled.
func (c *Config) GetMaxWidth() int {
	if c.Display.MaxWidth == 0 {
		return 110
	}
	return c.Display.MaxWidth
}
