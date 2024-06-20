package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/just-hms/brodo/execx"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"
)

// diff prints each todo created added in the current pr
//
// equivalent to: git reflog `git branch --show-current` | tail -n1 | awk '{print $1}' | xargs git diff
func diff(repo repo) error {
	out, err := execx.Command("git", "reflog", repo.curBranch).Output()
	if err != nil {
		return err
	}

	reflogLines := strings.Split(string(out), "\n")
	if len(reflogLines) == 0 {
		return err
	}

	branchCreationCommit := reflogLines[len(reflogLines)-2]
	firstCommitHash := strings.Fields(branchCreationCommit)[0]

	out, err = execx.Command("git", "diff", firstCommitHash).Output()
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
			fmt.Printf("%s:%d: %s\n", filename, lineno, line[1:])
		}

		if !strings.HasPrefix(line, "-") {
			lineno++
		}
	}
	return nil
}

type repo struct {
	owner     string
	name      string
	curBranch string
}

func info() (repo, error) {
	repo := repo{}

	out, err := execx.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return repo, err
	}
	repo.curBranch = strings.TrimSpace(string(out))

	out, err = execx.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return repo, err
	}

	remoteURL := strings.TrimSpace(string(out))
	var ownerRepo string

	if strings.HasPrefix(remoteURL, "https://github.com/") {
		ownerRepo = strings.TrimSuffix(remoteURL[len("https://github.com/"):], ".git")
	} else if strings.HasPrefix(remoteURL, "git@github.com:") {
		ownerRepo = strings.TrimSuffix(remoteURL[len("git@github.com:"):], ".git")
	} else {
		return repo, fmt.Errorf("unsupported remote URL format: %q", ownerRepo)
	}

	ownerRepoParts := strings.Split(ownerRepo, "/")
	if len(ownerRepoParts) < 2 {
		return repo, fmt.Errorf("repo url wrong format: %q", ownerRepo)
	}

	repo.owner, repo.name = ownerRepoParts[0], ownerRepoParts[1]

	return repo, nil
}

// fetchPrsNo fetches all the pr linked to the current branch
func fetchPrsNo(repo repo) ([]int, error) {
	out, err := execx.Command(
		"gh", "api",
		"-H", "Accept:application/vnd.github+json",
		fmt.Sprintf("/repos/%s/%s/pulls?head=%s:%s", repo.owner, repo.name, repo.owner, repo.curBranch),
	).Output()

	if err != nil {
		return nil, err
	}
	raw := gjson.GetBytes(out, `#.number`).Raw

	var prs []int
	err = json.Unmarshal([]byte(raw), &prs)
	if err != nil {
		return nil, fmt.Errorf("err: %s, response is malformed: %q", err, string(out))
	}
	return prs, nil
}

//go:embed comments.gql
var qComments string

// unresolved print all unresolved conversation in the specified pr
func unresolved(repo repo, pr int) error {
	out, err := execx.Command(
		"gh", "api", "graphql",
		"-f", fmt.Sprintf("owner=%s", repo.owner),
		"-f", fmt.Sprintf("repo=%s", repo.name),
		"-F", fmt.Sprintf("pr=%d", pr),
		"-f", fmt.Sprintf("query=%s", qComments),
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
		return fmt.Errorf("err: %s, response is malformed: %q", err, string(out))
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

func gh(repo repo) error {
	prsno, err := fetchPrsNo(repo)
	if err != nil {
		return err
	}

	for _, pr := range prsno {
		err := unresolved(repo, pr)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	r, err := info()
	if err != nil {
		log.Fatal(err)
	}

	fs := []func(repo) error{
		gh,
		diff,
	}

	var wg errgroup.Group
	for _, f := range fs {
		wg.Go(func() error { return f(r) })
	}
	err = wg.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
