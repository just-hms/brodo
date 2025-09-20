package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/just-hms/brodo/execx"
)

type Info struct {
	Origin   Repo
	Upstream *Repo
	Branch   string
	Commit   string
}

type Repo struct {
	Owner string
	Name  string
}

func GetInfo() (*Info, error) {
	b := &Info{}

	{
		out, err := execx.Command("git", "branch", "--show-current").Output()
		if err != nil {
			return nil, err
		}
		b.Branch = strings.TrimSpace(string(out))
	}

	{
		out, err := execx.Command("git", "rev-parse", "HEAD").Output()
		if err != nil {
			return nil, err
		}
		b.Commit = strings.TrimSpace(string(out))
	}

	{
		out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
		if err != nil {
			return nil, err
		}
		// Parse the origin repository URL
		originRepo, err := parseRepoURL(string(out))
		if err != nil {
			return nil, err
		}
		b.Origin = originRepo
	}

	{
		// Try to get the upstream URL
		out, err := execx.Command("git", "config", "--get", "remote.upstream.url").Output()
		if err != nil {
			return b, nil
		}

		upstreamRepo, err := parseRepoURL(string(out))
		if err != nil {
			return nil, err
		}
		b.Upstream = &upstreamRepo
	}

	return b, nil
}

// parseRepoURL parses a GitHub repository URL and returns the owner and repository name.
func parseRepoURL(remoteURL string) (Repo, error) {
	var r Repo
	remoteURL = strings.TrimSpace(remoteURL)

	var prefix string
	switch {
	case strings.HasPrefix(remoteURL, "https://github.com/"):
		prefix = "https://github.com/"
	case strings.HasPrefix(remoteURL, "git@github.com:"):
		prefix = "git@github.com:"
	default:
		return r, fmt.Errorf("unsupported remote URL format: %q", remoteURL)
	}

	ownerRepo := strings.TrimSuffix(strings.TrimPrefix(remoteURL, prefix), ".git")
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return r, fmt.Errorf("invalid repo URL format: %q", remoteURL)
	}

	r.Owner, r.Name = parts[0], parts[1]
	return r, nil
}
