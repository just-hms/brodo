package execx

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

type Cmd struct {
	cmd *exec.Cmd
}

func Command(name string, arg ...string) *Cmd {
	return &Cmd{
		cmd: exec.Command(name, arg...),
	}
}

func (c *Cmd) Output() ([]byte, error) {
	var stderr bytes.Buffer
	c.cmd.Stderr = &stderr

	out, err := c.cmd.Output()
	if err != nil {
		if stderr.Len() == 0 {
			return out, err
		}
		return out, errors.New(strings.ReplaceAll(stderr.String(), "\n", "\\n"))
	}

	return out, nil
}
