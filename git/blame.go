package git

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/just-hms/brodo/execx"
)

func Blame(b *Info, file string, line uint32) (string, error) {
	cmd := execx.Command("git", "blame", fmt.Sprintf("-L%d,%d", line, line), "--", file)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	if bytes.Contains(out, []byte("Not Committed Yet")) {
		out, err = execx.Command("git", "config", "user.name").Output()
		return strings.TrimSpace(string(out)), err
	}

	blameLine := string(out)
	start := strings.Index(blameLine, "(")
	end := strings.Index(blameLine, ")")
	if start != -1 && end != -1 && end > start {
		fields := strings.Fields(blameLine[start+1 : end])
		if len(fields) > 0 {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("couldn't blame: %s", blameLine)
}
