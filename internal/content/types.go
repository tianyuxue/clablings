// Package content 解析 clablings 题库（清单 / 章节 / 题目）。
package content

// Manifest 对应题库根 clablings.yaml。
type Manifest struct {
	Version  int      `yaml:"version"`
	Name     string   `yaml:"name"`
	Chapters []string `yaml:"chapters"`
}

// ExerciseType 是题型。
type ExerciseType string

const (
	TypeConfigFix     ExerciseType = "config-fix"
	TypeCodeImplement ExerciseType = "code-implement"
)

// FileMapping 把宿主可编辑文件映射到节点目标路径。
type FileMapping struct {
	Source string `yaml:"source"`
	Node   string `yaml:"node"`
	Dest   string `yaml:"dest"`
}

// NodeCmd 是在指定节点内执行的命令。
type NodeCmd struct {
	Node string `yaml:"node"`
	Cmd  string `yaml:"cmd"`
}

// BuildSpec 描述 C 题的构建方式（本分册仅解析）。
type BuildSpec struct {
	Image   string `yaml:"image"`
	Command string `yaml:"command"`
	Output  string `yaml:"output"`
}

// RunSpec 描述 C 题二进制的下发与进程管理（本分册仅解析）。
type RunSpec struct {
	Node  string `yaml:"node"`
	Dest  string `yaml:"dest"`
	Stop  string `yaml:"stop"`
	Start string `yaml:"start"`
}

// Deploy 描述章节部署参数。
type Deploy struct {
	Timeout string `yaml:"timeout"`
}

// Assertion 是单条断言声明。
type Assertion struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"` // exec / file / interface / link
	Layer   string `yaml:"layer"`
	Node    string `yaml:"node"`
	Timeout string `yaml:"timeout"`
	Hint    string `yaml:"hint"`

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

// TextMatch 描述对文本的匹配方式；三者互斥，按 Equals > Contains > Regex 优先。
type TextMatch struct {
	Equals   string `yaml:"equals"`
	Contains string `yaml:"contains"`
	Regex    string `yaml:"regex"`
}

// Match 描述 exec 断言的期望。JSON 非空时按 JSONPath 取值后用内联 TextMatch 匹配。
type Match struct {
	ExitCode  *int       `yaml:"exit_code"`
	JSON      string     `yaml:"json"`
	TextMatch `yaml:",inline"`
	Stdout    *TextMatch `yaml:"stdout"`
}

// Chapter 对应 chapter.yaml 及其目录。
type Chapter struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Topology string   `yaml:"topology"`
	Deploy   Deploy   `yaml:"deploy"`
	Setup    []string `yaml:"setup"`
	Teardown []string `yaml:"teardown"`

	Dir          string      `yaml:"-"`
	TopologyPath string      `yaml:"-"`
	TopologyName string      `yaml:"-"` // 拓扑顶层 name，用于推导容器长名
	Exercises    []*Exercise `yaml:"-"`
}

// Exercise 对应 exercise.yaml 及其目录。
type Exercise struct {
	ID         string        `yaml:"id"`
	Title      string        `yaml:"title"`
	Type       ExerciseType  `yaml:"type"`
	Timeout    string        `yaml:"timeout"`
	Files      []FileMapping `yaml:"files"`
	Reload     []NodeCmd     `yaml:"reload"`
	Build      *BuildSpec    `yaml:"build"`
	Run        *RunSpec      `yaml:"run"`
	Assertions []Assertion   `yaml:"assertions"`

	Dir     string `yaml:"-"`
	Chapter string `yaml:"-"`
}

// FullID 返回全限定 ID：chapter-id/exercise-id。
func (e *Exercise) FullID() string { return e.Chapter + "/" + e.ID }

// Library 是解析后的完整题库。
type Library struct {
	Root     string
	Manifest Manifest
	Chapters []*Chapter
	ByID     map[string]*Exercise
	Order    []*Exercise
}
