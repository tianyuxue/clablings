package content

import "testing"

func TestInstanceIDFromEnv(t *testing.T) {
	t.Setenv("CLABLINGS_INSTANCE", "Team A")
	if got := InstanceID("/whatever"); got != "team-a" {
		t.Fatalf("InstanceID = %q, want team-a", got)
	}
}

func TestInstanceIDFromRootHash(t *testing.T) {
	t.Setenv("CLABLINGS_INSTANCE", "")
	a := InstanceID("/a")
	b := InstanceID("/a")
	c := InstanceID("/b")
	if len(a) != 8 {
		t.Fatalf("len(%q) = %d, want 8", a, len(a))
	}
	if a != b {
		t.Fatalf("same root should be stable: %q != %q", a, b)
	}
	if a == c {
		t.Fatalf("different roots should differ: %q == %q", a, c)
	}
}

func TestInstanceIDFromEnvAllInvalid(t *testing.T) {
	t.Setenv("CLABLINGS_INSTANCE", "!!!")
	got := InstanceID("/a")
	if len(got) != 8 {
		t.Fatalf("InstanceID = %q, want 8-char root hash", got)
	}
}

func TestLabNameSanitizesInstance(t *testing.T) {
	got := LabName("/root", "!!!", "x")
	if got != "root-0-x" {
		t.Fatalf("LabName = %q, want root-0-x", got)
	}
}

func TestLabNameSanitizes(t *testing.T) {
	got := LabName("/home/me/My Labs/", "abcd1234", "Clablings OSPF")
	want := "my-labs-abcd1234-clablings-ospf"
	if got != want {
		t.Fatalf("LabName = %q, want %q", got, want)
	}
}

func TestLabNameFallsBackWhenBasenameEmpty(t *testing.T) {
	got := LabName("/", "abcd1234", "x")
	if got != "clablings-abcd1234-x" {
		t.Fatalf("LabName = %q", got)
	}
}
