package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLibrary(t *testing.T) {
	root := filepath.Join("testdata", "library")
	lib, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lib.Manifest.Version != 1 {
		t.Fatalf("version = %d", lib.Manifest.Version)
	}
	if len(lib.Chapters) != 2 {
		t.Fatalf("chapters = %d, 期望 2", len(lib.Chapters))
	}
	if len(lib.Order) != 2 {
		t.Fatalf("order = %d, 期望 2", len(lib.Order))
	}

	c0 := lib.Chapters[0]
	if c0.ID != "01-ospf-mtu" || c0.Title != "OSPF MTU 不匹配" {
		t.Fatalf("chapter0 = %+v", c0)
	}
	if c0.TopologyPath != filepath.Join(root, "chapters", "01-ospf-mtu", "topology.clab.yml") {
		t.Fatalf("topology path = %q", c0.TopologyPath)
	}
	if c0.TopologyName != "clablings-ospf-mtu" {
		t.Fatalf("topology name = %q", c0.TopologyName)
	}
	if len(c0.Exercises) != 1 {
		t.Fatalf("exercises = %d", len(c0.Exercises))
	}

	ex := lib.ByID["01-ospf-mtu/01_mtu"]
	if ex == nil {
		t.Fatal("ByID 缺少 01-ospf-mtu/01_mtu")
	}
	if ex.Type != TypeConfigFix {
		t.Fatalf("type = %q", ex.Type)
	}
	if len(ex.Files) != 1 || ex.Files[0].Source != "frr1/frr.conf" || ex.Files[0].Node != "r1" {
		t.Fatalf("files = %+v", ex.Files)
	}
	if len(ex.Assertions) != 1 || ex.Assertions[0].Type != "exec" {
		t.Fatalf("assertions = %+v", ex.Assertions)
	}
	if ex.Assertions[0].Match == nil || ex.Assertions[0].Match.JSON != "$.neighbors.*.state" {
		t.Fatalf("match = %+v", ex.Assertions[0].Match)
	}
	if ex.Assertions[0].Match.Equals != "Full" {
		t.Fatalf("inline equals = %q", ex.Assertions[0].Match.Equals)
	}
	if ex.Dir != filepath.Join(root, "chapters", "01-ospf-mtu", "exercises", "01_mtu") {
		t.Fatalf("exercise dir = %q", ex.Dir)
	}

	code := lib.ByID["02-arp/01_arp"]
	if code == nil || code.Build == nil || code.Run == nil {
		t.Fatalf("code exercise = %+v", code)
	}
	if code.Build.Image != "gcc:13" || code.Build.Output != "arp_responder" {
		t.Fatalf("build = %+v", code.Build)
	}
	if len(code.Assertions) != 1 || code.Assertions[0].Match == nil || code.Assertions[0].Match.Stdout == nil {
		t.Fatalf("assertions = %+v", code.Assertions)
	}
	if code.Assertions[0].Match.Stdout.Contains != "reply from" {
		t.Fatalf("stdout contains = %q", code.Assertions[0].Match.Stdout.Contains)
	}
}

func TestLoadToleratesMissingExercisesDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "clablings.yaml"), "version: 1\nchapters:\n  - chapters/empty\n")
	writeFile(t, filepath.Join(root, "chapters", "empty", "chapter.yaml"), "id: empty\ntitle: t\ntopology: topology.clab.yml\n")
	writeFile(t, filepath.Join(root, "chapters", "empty", "topology.clab.yml"), "name: lab-empty\ntopology:\n  nodes: {}\n")

	lib, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lib.Chapters) != 1 || len(lib.Chapters[0].Exercises) != 0 {
		t.Fatalf("chapters = %+v", lib.Chapters)
	}
}

func TestLoadErrors(t *testing.T) {
	validManifest := "version: 1\nchapters:\n  - chapters/c1\n"
	validChapter := "id: c1\ntitle: t\ntopology: topology.clab.yml\n"
	validTopology := "name: lab-c1\ntopology:\n  nodes: {}\n"

	tests := []struct {
		name  string
		want  string
		setup func(t *testing.T, root string)
	}{
		{
			name: "manifest 缺少 version",
			want: "缺少 version",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), "chapters:\n  - chapters/c1\n")
				writeFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), validChapter)
				writeFile(t, filepath.Join(root, "chapters", "c1", "topology.clab.yml"), validTopology)
			},
		},
		{
			name: "manifest chapters 为空",
			want: "chapters 为空",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), "version: 1\nchapters: []\n")
			},
		},
		{
			name: "chapter 缺少 id",
			want: "缺少 id",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), validManifest)
				writeFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), "title: t\ntopology: topology.clab.yml\n")
				writeFile(t, filepath.Join(root, "chapters", "c1", "topology.clab.yml"), validTopology)
			},
		},
		{
			name: "chapter 缺少 topology",
			want: "缺少 topology",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), validManifest)
				writeFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), "id: c1\ntitle: t\n")
			},
		},
		{
			name: "topology 文件不存在",
			want: "topology 文件不存在",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), validManifest)
				writeFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), validChapter)
			},
		},
		{
			name: "topology 缺少顶层 name",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), validManifest)
				writeFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), validChapter)
				writeFile(t, filepath.Join(root, "chapters", "c1", "topology.clab.yml"), "topology:\n  nodes: {}\n")
			},
		},
		{
			name: "exercise 非法 type",
			want: "非法 type",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), validManifest)
				writeFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), validChapter)
				writeFile(t, filepath.Join(root, "chapters", "c1", "topology.clab.yml"), validTopology)
				writeFile(t, filepath.Join(root, "chapters", "c1", "exercises", "ex1", "exercise.yaml"), "id: ex1\ntype: bogus\n")
			},
		},
		{
			name: "code-implement 缺少 build/run",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), validManifest)
				writeFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), validChapter)
				writeFile(t, filepath.Join(root, "chapters", "c1", "topology.clab.yml"), validTopology)
				writeFile(t, filepath.Join(root, "chapters", "c1", "exercises", "ex1", "exercise.yaml"), "id: ex1\ntype: code-implement\n")
			},
		},
		{
			name: "重复的题目 ID",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "clablings.yaml"), "version: 1\nchapters:\n  - chapters/c1\n  - chapters/c2\n")
				for _, dir := range []string{"c1", "c2"} {
					writeFile(t, filepath.Join(root, "chapters", dir, "chapter.yaml"), "id: dup\ntitle: t\ntopology: topology.clab.yml\n")
					writeFile(t, filepath.Join(root, "chapters", dir, "topology.clab.yml"), "name: lab-"+dir+"\ntopology:\n  nodes: {}\n")
					writeFile(t, filepath.Join(root, "chapters", dir, "exercises", "ex1", "exercise.yaml"), "id: ex1\ntype: config-fix\n")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)
			_, err := Load(root)
			if err == nil {
				t.Fatal("Load 期望返回错误，实际为 nil")
			}
			if tt.want != "" && !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, 期望包含 %q", err, tt.want)
			}
		})
	}
}

func writeFile(t *testing.T, path, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
