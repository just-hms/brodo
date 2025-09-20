package git

import (
	"github.com/just-hms/brodo/execx"
)

func Diff(b *Info, against string) (string, error) {
	out, err := execx.Command("git", "diff", against).Output()
	return string(out), err
}
