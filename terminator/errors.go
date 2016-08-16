package terminator

import "fmt"

type Error struct {
	args args
}

func (err Error) Error() string {
	return fmt.Sprintf("%+v", err.args)
}
