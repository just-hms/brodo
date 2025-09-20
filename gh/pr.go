package gh

import (
	"encoding/json"
	"fmt"

	"github.com/just-hms/brodo/execx"
	"github.com/just-hms/brodo/git"
	"github.com/tidwall/gjson"
)

// FetchPrs fetches all the pr linked to the current branch
func FetchPrs(i *git.Info) ([]*PR, error) {
	repos := []git.Repo{i.Origin}
	if i.Upstream != nil {
		repos = append(repos, *i.Upstream)
	}

	var res []*PR

	for _, repo := range repos {
		out, err := execx.Command(
			"gh", "api",
			"-H", "Accept:application/vnd.github+json",
			fmt.Sprintf("/repos/%s/%s/pulls?head=%s:%s", repo.Owner, repo.Name, i.Origin.Owner, i.Branch),
		).Output()
		if err != nil {
			return nil, err
		}

		raw := gjson.GetBytes(out, `#.{number,base.ref,base.sha}`).Raw

		var prs []*PR
		if err := json.Unmarshal([]byte(raw), &prs); err != nil {
			return nil, fmt.Errorf("err: %s, response is malformed: %q", err, string(out))
		}

		for _, pr := range prs {
			pr.Repo = repo
		}

		res = append(res, prs...)
	}

	return res, nil

}
