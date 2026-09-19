// Package content parses the clablings library (manifest / chapters / exercises).
package content

// Manifest corresponds to the library root clablings.yaml.
type Manifest struct {
	Version  int      `yaml:"version"`
	Name     string   `yaml:"name"`
	Chapters []string `yaml:"chapters"`
}

// ExerciseType is the exercise type.
type ExerciseType string

const (
	TypeConfigFix     ExerciseType = "config-fix"
	TypeCodeImplement ExerciseType = "code-implement"
)

// FileMapping maps an editable host file to a node target path.
type FileMapping struct {
	Source string `yaml:"source"`
	Node   string `yaml:"node"`
	Dest   string `yaml:"dest"`
}

// NodeCmd is a command executed inside a specific node.
type NodeCmd struct {
	Node string `yaml:"node"`
	Cmd  string `yaml:"cmd"`
}

// BuildSpec describes how a C exercise is built (parsed only in this volume).
type BuildSpec struct {
	Image   string `yaml:"image"`
	Command string `yaml:"command"`
	Output  string `yaml:"output"`
}

// RunSpec describes delivery and process management of a C exercise binary (parsed only in this volume).
type RunSpec struct {
	Node  string `yaml:"node"`
	Dest  string `yaml:"dest"`
	Stop  string `yaml:"stop"`
	Start string `yaml:"start"`
}

// Deploy describes chapter deployment parameters.
type Deploy struct {
	Timeout string `yaml:"timeout"`
}

// Assertion is a single assertion declaration.
type Assertion struct {
	Name    string `yaml:"-"`
	Type    string `yaml:"type"` // exec / file / interface / link
	Layer   string `yaml:"layer"`
	Node    string `yaml:"node"`
	Timeout string `yaml:"timeout"`
	Wait    string `yaml:"wait"`
	Hint    string `yaml:"-"`

	// exec
	Cmd   string `yaml:"cmd"`
	Match *Match `yaml:"match"`

	// file
	Source      string `yaml:"source"`
	Path        string `yaml:"path"`
	Contains    string `yaml:"contains"`
	NotContains string `yaml:"not_contains"`

	// interface / link
	Interface string `yaml:"interface"`
	State     string `yaml:"state"`
	A         string `yaml:"a"`
	PortA     string `yaml:"port_a"`
	B         string `yaml:"b"`
	PortB     string `yaml:"port_b"`
}

// TextMatch describes how text is matched; the three are mutually exclusive, preferring Equals > Contains > Regex.
type TextMatch struct {
	Equals   string `yaml:"equals"`
	Contains string `yaml:"contains"`
	Regex    string `yaml:"regex"`
}

// Match describes the expectation of an exec assertion. When JSON is non-empty, a JSONPath value is
// extracted and matched with the inline TextMatch.
type Match struct {
	ExitCode  *int   `yaml:"exit_code"`
	JSON      string `yaml:"json"`
	TextMatch `yaml:",inline"`
	Stdout    *TextMatch `yaml:"stdout"`
}

// Chapter corresponds to chapter.yaml and its directory.
type Chapter struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"-"`
	Topology string   `yaml:"topology"`
	Deploy   Deploy   `yaml:"deploy"`
	Setup    []string `yaml:"setup"`
	Teardown []string `yaml:"teardown"`

	Difficulty       int      `yaml:"-"`
	Tags             []string `yaml:"-"`
	EstimatedMinutes int      `yaml:"-"`
	Prerequisites    []string `yaml:"-"`
	Objectives       []string `yaml:"-"`

	Dir          string      `yaml:"-"`
	TopologyPath string      `yaml:"-"`
	TopologyName string      `yaml:"-"` // topology top-level name, used to derive container long names
	LabName      string      `yaml:"-"` // instance-derived containerlab lab name
	Exercises    []*Exercise `yaml:"-"`
}

// HasTopology reports whether the chapter declares a topology. TopologyPath
// and TopologyName are set together by loadChapter when chapter.yaml has a
// non-empty topology field.
func (c *Chapter) HasTopology() bool { return c.TopologyPath != "" }

// Exercise corresponds to exercise.yaml and its directory.
type Exercise struct {
	ID         string        `yaml:"id"`
	Title      string        `yaml:"-"`
	Type       ExerciseType  `yaml:"type"`
	Timeout    string        `yaml:"timeout"`
	Files      []FileMapping `yaml:"files"`
	Reload     []NodeCmd     `yaml:"reload"`
	Hints      []string      `yaml:"-"`
	Build      *BuildSpec    `yaml:"build"`
	Run        *RunSpec      `yaml:"run"`
	Assertions []Assertion   `yaml:"-"`

	Difficulty int      `yaml:"-"`
	Tags       []string `yaml:"-"`

	Dir        string   `yaml:"-"`
	Chapter    string   `yaml:"-"`
	ChapterRef *Chapter `yaml:"-"`
}

// FullID returns the fully qualified ID: chapter-id/exercise-id.
func (e *Exercise) FullID() string { return e.Chapter + "/" + e.ID }

// Library is the fully parsed library.
type Library struct {
	Root     string
	Manifest Manifest
	Chapters []*Chapter
	ByID     map[string]*Exercise
	Order    []*Exercise
}

// ChapterByID returns the chapter with the given ID, or nil.
func (l *Library) ChapterByID(id string) *Chapter {
	for _, ch := range l.Chapters {
		if ch.ID == id {
			return ch
		}
	}
	return nil
}
