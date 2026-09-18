package content

import "testing"

func TestLibraryResolve(t *testing.T) {
	ch1 := &Chapter{ID: "c1"}
	e1 := &Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch1}
	e2 := &Exercise{ID: "e2", Chapter: "c1", ChapterRef: ch1}
	lib := &Library{Order: []*Exercise{e1, e2}, ByID: map[string]*Exercise{"c1/e1": e1, "c1/e2": e2}}

	if got, err := lib.Resolve("c1/e1"); err != nil || got != e1 {
		t.Fatalf("Resolve(c1/e1) = %v, %v", got, err)
	}
	if got, err := lib.Resolve("e2"); err != nil || got != e2 {
		t.Fatalf("Resolve(e2) = %v, %v", got, err)
	}
	if _, err := lib.Resolve("nope"); err == nil {
		t.Fatal("expected unknown error")
	}

	chA := &Chapter{ID: "a"}
	chB := &Chapter{ID: "b"}
	d1 := &Exercise{ID: "dup", Chapter: "a", ChapterRef: chA}
	d2 := &Exercise{ID: "dup", Chapter: "b", ChapterRef: chB}
	dup := &Library{Order: []*Exercise{d1, d2}, ByID: map[string]*Exercise{"a/dup": d1, "b/dup": d2}}
	if _, err := dup.Resolve("dup"); err == nil {
		t.Fatal("expected ambiguous error")
	}
}
