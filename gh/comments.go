package gh

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/just-hms/brodo/execx"
	"github.com/just-hms/brodo/git"
	"github.com/tidwall/gjson"
)

type PR struct {
	No  int    `json:"number"`
	Ref string `json:"ref"`
	Sha string `json:"sha"`

	Repo git.Repo `json:"-"`
}

//go:embed queries/comments.gql
var qComments string

// Unresolved print all Unresolved conversation in the specified pr
func Unresolved(pr *PR) error {
	out, err := execx.Command(
		"gh", "api", "graphql",
		"-f", fmt.Sprintf("owner=%s", pr.Repo.Owner),
		"-f", fmt.Sprintf("repo=%s", pr.Repo.Name),
		"-F", fmt.Sprintf("pr=%d", pr.No),
		"-f", fmt.Sprintf("query=%s", qComments),
	).Output()
	if err != nil {
		log.Fatal(err)
		return err
	}

	raw := gjson.GetBytes(out, `data.repository.pullRequest.reviewThreads.edges.#.node.{isResolved,"comments":comments.nodes.#.{path,line,body,originalLine,"author":author.login}}`).Raw

	type thread struct {
		IsResolved bool
		Comments   []struct {
			Path         string
			Line         int
			OriginalLine int
			Body         string
			Author       string
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
		if topComment.Line != topComment.OriginalLine {
			fmt.Printf("[@%s] %s:outdated: %s\n", topComment.Author, topComment.Path, body)
		} else {
			fmt.Printf("[@%s] %s:%d: %s\n", topComment.Author, topComment.Path, topComment.Line, body)

		}
	}

	return nil
}
