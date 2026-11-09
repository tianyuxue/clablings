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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripSudoFlag(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("stripSudoFlag(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
