package content

import "fmt"

// Resolve returns the exercise with the given ID. id may be a fully qualified
// "chapter-id/exercise-id" or an exercise-local ID that is unique across the
// library. It returns an error for unknown or ambiguous IDs.
func (l *Library) Resolve(id string) (*Exercise, error) {
	if ex := l.ByID[id]; ex != nil {
		return ex, nil
	}
	var found *Exercise
	for _, ex := range l.Order {
		if ex.ID == id {
			if found != nil {
				return nil, fmt.Errorf("exercise ID %q is not unique; use chapter-id/exercise-id", id)
			}
			found = ex
		}
	}
	if found == nil {
		return nil, fmt.Errorf("unknown exercise %q", id)
	}
	return found, nil
}
