package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

func diff(against string) {
	out, err := exec.Command("git", "diff", against).Output()
	if err != nil {
		panic(err)
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
}

func branchrefs() {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		panic(err)
	}

	currentBranch := strings.TrimSpace(string(out))
	currentBranchNo := strings.Split(currentBranch, "-")[0]

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
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
}

func prs() (string, string, []int) {
	out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		panic(err)
	}

	remoteURL := strings.TrimSpace(string(out))
	var ownerRepo string

	if strings.HasPrefix(remoteURL, "https://github.com/") {
		ownerRepo = strings.TrimSuffix(remoteURL[len("https://github.com/"):], ".git")
	} else if strings.HasPrefix(remoteURL, "git@github.com:") {
		ownerRepo = strings.TrimSuffix(remoteURL[len("git@github.com:"):], ".git")
	} else {
		panic("Unsupported remote URL format")
	}

	ownerRepoParts := strings.Split(ownerRepo, "/")
	owner := ownerRepoParts[0]
	repo := ownerRepoParts[1]

	out, err = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		panic(err)
	}

	branch := strings.TrimSpace(string(out))

	out, err = exec.Command(
		"gh", "api",
		"-H", "Accept:application/vnd.github+json",
		fmt.Sprintf("/repos/%s/%s/pulls?head=%s:%s", owner, repo, owner, branch),
	).Output()
	if err != nil {
		panic(err)
	}

	res := gjson.Get(string(out), "#.number")

	// todo: add check
	var prnos []int
	for _, pri := range res.Array() {
		prnos = append(prnos, int(pri.Int()))
	}

	return owner, repo, prnos
}

//go:embed query.gql
var query string

func unresolved(owner, repo string, pr int) {
	out, err := exec.Command(
		"gh", "api", "graphql",
		"-f", fmt.Sprintf("owner=%s", owner),
		"-f", fmt.Sprintf("repo=%s", repo),
		"-F", fmt.Sprintf("pr=%d", pr),
		"-f", fmt.Sprintf("query=%s", query),
	).Output()
	if err != nil {
		panic(err)
	}

	threads := gjson.Get(string(out), "data.repository.pullRequest.reviewThreads.edges.#.node")

	for _, thread := range threads.Array() {
		if thread.Get("isResolved").Bool() {
			continue
		}

		comment := thread.Get("comments.0.nodes.0")
		filename := comment.Get("path").Str
		lineno := comment.Get("line").Float()
		body := strings.ReplaceAll(comment.Get("body").Str, "\n", "\\n")
		user := comment.Get("author.login").Str

		fmt.Printf("[%s]%s:%d: %s\n", user, filename, int(lineno), body)
	}
}

func main() {
	// todo: find the current branch
	diff("origin/develop")
	branchrefs()
	owner, repo, prs := prs()
	for _, pr := range prs {
		unresolved(owner, repo, pr)
	}
}
