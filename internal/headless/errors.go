package headless

import "fmt"

func errUnknownExercise(id string) error {
	return fmt.Errorf("unknown exercise %s", id)
}
