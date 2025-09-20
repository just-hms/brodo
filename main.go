package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"slices"

	"github.com/just-hms/brodo/gh"
	"github.com/just-hms/brodo/git"
	"github.com/just-hms/brodo/sit"
	"golang.org/x/sync/errgroup"
)

var fPattern = flag.String("pattern", `(?i)TODO`, "pass down which pattern to use (default will search for all cases TODO)")

func main() {
	flag.Parse()
	pattern, err := regexp.Compile(*fPattern)
	if err != nil {
		log.Fatal(err)
	}
	info, err := git.GetInfo()
	if err != nil {
		log.Fatal(err)
	}

	prs, err := gh.FetchPrs(info)
	if err != nil {
		log.Fatal(err)
	}

	var wg errgroup.Group
	wg.SetLimit(10)

	for _, pr := range prs {
		wg.Go(func() error { return gh.Unresolved(pr) })
	}

	prs = slices.Clone(prs)
	if len(prs) == 0 {
		if len(flag.Args()) == 0 {
			log.Fatal("No PR detected, pass in an argument specifying the branch you want to diff against")
		}
		prs = append(prs, &gh.PR{
			Sha: flag.Arg(0),
		})
	}

	for _, pr := range prs {
		wg.Go(func() error {
			diff, err := git.Diff(info, pr.Sha)
			if err != nil {
				log.Printf("Got: %v, try to run `git fetch` to solve the issue\n", err)
				return nil
			}

			// TODO: if treesitter is added, filter out files entirely if the pattern never matches

			for file, additions := range git.Additions(diff) {
				todoAdditions := []git.Addition{}
				for _, add := range additions {
					m := pattern.FindIndex([]byte(add.Content))
					if m == nil {
						continue
					}
					add.Column = uint32(m[0])
					todoAdditions = append(todoAdditions, add)
				}

				if len(todoAdditions) == 0 {
					continue
				}

				comments, err := sit.Comments(file)
				if err != nil {
					return err
				}

				todoAdditions = slices.DeleteFunc(todoAdditions, func(add git.Addition) bool {
					return !slices.ContainsFunc(comments, func(c *sit.Range) bool { return c.Contains(sit.Point(add.Point)) })
				})

				// TODO: ideally git blame for todos
				for _, add := range todoAdditions {
					fmt.Printf("[TODO] %s:%d %s\n", file, add.Row+1, add.Content)
				}
			}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		log.Fatal(err)
	}
}
