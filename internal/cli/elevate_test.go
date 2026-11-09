package cli

import (
	"reflect"
	"testing"
)

func TestStripSudoFlag(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"long flag", []string{"watch", "--sudo"}, []string{"watch"}},
		{"equals true", []string{"--sudo=true", "test", "--all"}, []string{"test", "--all"}},
		{"between args", []string{"debug", "--sudo", "--root", "/x"}, []string{"debug", "--root", "/x"}},
		{"no flag", []string{"list", "--root=/x"}, []string{"list", "--root=/x"}},
		{"similar prefix", []string{"--sudolike", "x"}, []string{"--sudolike", "x"}},
		{"after delimiter", []string{"run", "--", "--sudo"}, []string{"run", "--", "--sudo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripSudoFlag(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("stripSudoFlag(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestShouldElevate(t *testing.T) {
	tests := []struct {
		name string
		args []string
		euid int
		want bool
	}{
		{"before subcommand", []string{"--sudo", "watch"}, 1000, true},
		{"after subcommand", []string{"watch", "--sudo"}, 1000, false},
		{"before debug", []string{"--sudo", "debug"}, 1000, true},
		{"after debug", []string{"debug", "--sudo"}, 1000, false},
		{"equals true", []string{"--sudo=true", "test"}, 1000, true},
		{"equals false", []string{"--sudo=false", "test"}, 1000, false},
		{"already root", []string{"--sudo", "watch"}, 0, false},
		{"after delimiter", []string{"run", "--", "--sudo"}, 1000, false},
		{"empty", nil, 1000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldElevate(tt.args, tt.euid); got != tt.want {
				t.Fatalf("shouldElevate(%v, %d) = %v, want %v", tt.args, tt.euid, got, tt.want)
			}
		})
	}
}

func TestSudoArgv(t *testing.T) {
	argv := sudoArgv("/usr/local/bin/clablings", []string{"watch", "--sudo"}, map[string]string{
		"CLABLINGS_ROOT":     "/lib",
		"CLABLINGS_INSTANCE": "inst",
		"CLABLINGS_LANG":     "en",
	})
	want := []string{
		"sudo",
		"CLABLINGS_ROOT=/lib",
		"CLABLINGS_INSTANCE=inst",
		"CLABLINGS_LANG=en",
		"/usr/local/bin/clablings",
		"watch",
	}
	if !reflect.DeepEqual(argv, want) {
		t.Fatalf("sudoArgv = %v, want %v", argv, want)
	}

	argv = sudoArgv("/bin/clablings", []string{"version"}, map[string]string{
		"CLABLINGS_ROOT": "",
		"CLABLINGS_LANG": "zh",
	})
	want = []string{"sudo", "CLABLINGS_LANG=zh", "/bin/clablings", "version"}
	if !reflect.DeepEqual(argv, want) {
		t.Fatalf("sudoArgv = %v, want %v", argv, want)
	}
}
