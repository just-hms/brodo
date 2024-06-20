package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"
)

// git reflog `git branch --show-current` | tail -n1 | awk '{print $1}' | xargs git diff
func diff() error {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return err
	}

	branch := strings.TrimSpace(string(out))

	out, err = exec.Command("git", "reflog", branch).Output()
	if err != nil {
		return err
	}

	reflogLines := strings.Split(string(out), "\n")
	if len(reflogLines) == 0 {
		return errors.New("no reflog entries found")
	}

	branchCreationCommit := reflogLines[len(reflogLines)-2]
	firstCommitHash := strings.Fields(branchCreationCommit)[0]

	out, err = exec.Command("git", "diff", firstCommitHash).Output()
	if err != nil {
		return err
	}

	diffOutput := string(out)

	var (
		filename string
		lineno   int
	)

	for _, line := range strings.Split(diffOutput, "\n") {
		if strings.HasPrefix(line, "diff --git a/") {
			filename = strings.Split(line, " ")[2][2:]
			continue
		}

		if strings.HasPrefix(line, "@@ ") {
			parts := strings.Split(strings.Split(line, "+")[1], ",")
			lineno, err = strconv.Atoi(parts[0])
			if err != nil {
				lineno = 0
			}
			continue
		}

		if strings.HasPrefix(line, "+") && (strings.Contains(line, "TODO") || strings.Contains(line, "todo")) {
			content := strings.TrimSpace(line[1:])
			fmt.Printf("%s:%d: %s\n", filename, lineno, content)
		}

		if !strings.HasPrefix(line, "-") {
			lineno++
		}
	}
	return nil
}

func branchrefs() error {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return err
	}

	currentBranch := strings.TrimSpace(string(out))
	currentBranchNo := strings.Split(currentBranch, "-")[0]

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// skipping hidden dirs
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// skipping hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		lines, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for lineno, line := range strings.Split(string(lines), "\n") {
			if strings.Contains(line, currentBranchNo) {
				fmt.Printf("%s:%d: %s\n", path, lineno+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	return nil
}

type repo struct {
	owner string
	name  string
}

func fetchRepo() (repo, error) {
	out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return repo{}, err
	}

	remoteURL := strings.TrimSpace(string(out))
	var ownerRepo string

	if strings.HasPrefix(remoteURL, "https://github.com/") {
		ownerRepo = strings.TrimSuffix(remoteURL[len("https://github.com/"):], ".git")
	} else if strings.HasPrefix(remoteURL, "git@github.com:") {
		ownerRepo = strings.TrimSuffix(remoteURL[len("git@github.com:"):], ".git")
	} else {
		return repo{}, fmt.Errorf("unsupported remote URL format: %q", ownerRepo)
	}

	ownerRepoParts := strings.Split(ownerRepo, "/")
	if len(ownerRepoParts) < 2 {
		return repo{}, fmt.Errorf("repo url wrong format: %q", ownerRepo)
	}

	return repo{
		owner: ownerRepoParts[0],
		name:  ownerRepoParts[1],
	}, nil
}
func prs(owner, repo string) ([]int, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return nil, err
	}

	branch := strings.TrimSpace(string(out))

	out, err = exec.Command(
		"gh", "api",
		"-H", "Accept:application/vnd.github+json",
		fmt.Sprintf("/repos/%s/%s/pulls?head=%s:%s", owner, repo, owner, branch),
	).Output()
	if err != nil {
		return nil, err
	}

	raw := gjson.GetBytes(out, "#.number").Raw

	var prnos []int
	err = json.Unmarshal([]byte(raw), &prnos)
	if err != nil {
		return nil, err
	}
	return prnos, nil
}

//go:embed query.gql
var query string

func unresolved(owner, repo string, pr int) error {
	out, err := exec.Command(
		"gh", "api", "graphql",
		"-f", fmt.Sprintf("owner=%s", owner),
		"-f", fmt.Sprintf("repo=%s", repo),
		"-F", fmt.Sprintf("pr=%d", pr),
		"-f", fmt.Sprintf("query=%s", query),
	).Output()
	if err != nil {
		return err
	}

	raw := gjson.GetBytes(out, `data.repository.pullRequest.reviewThreads.edges.#.node.{isResolved,"comments":comments.nodes.#.{"filename":path,"lineno":line,body,"author":author.login}}`).Raw

	type thread struct {
		IsResolved bool
		Comments   []struct {
			Filename string
			Lineno   int
			Body     string
			Author   string
		}
	}

	threads := []thread{}
	err = json.Unmarshal([]byte(raw), &threads)
	if err != nil {
		return fmt.Errorf("response is malformed: %q", string(out))
	}

	for _, thread := range threads {
		if thread.IsResolved {
			continue
		}
		if len(thread.Comments) == 0 {
			continue
		}
		topComment := thread.Comments[0]
		body := strings.ReplaceAll(topComment.Body, "\n", "\\n")
		fmt.Printf("[%s] %s:%d: %s\n", topComment.Author, topComment.Filename, topComment.Lineno, body)
	}

	return nil
}

func gh() error {
	repo, err := fetchRepo()
	if err != nil {
		return err
	}
	prs, err := prs(repo.owner, repo.name)
	if err != nil {
		return err
	}
	for _, pr := range prs {
		err := unresolved(repo.owner, repo.name, pr)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	fs := []func() error{
		gh,
		diff,
		branchrefs,
	}

	var wg errgroup.Group
	for _, f := range fs {
		wg.Go(f)
	}
	err := wg.Wait()
	if err != nil {
		fmt.Println(err)
	}
}
