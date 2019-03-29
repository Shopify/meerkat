package repos

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"os"
)

type GithubRepo struct {
	r *github.Repository
}

func NewGithubRepo() error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "05303fba719627b0853c53b93a24de679938ef5f"},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// list all repositories for the authenticated user
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 1000},
	}
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, "Shopify", opt)
		if err != nil {
			return err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
		fmt.Println("appended:", len(repos), " total:", len(allRepos))
	}
	for _, r := range allRepos {
		fmt.Println(*r.CloneURL)
	}
	return nil
}

func CloneToDisk() error {
	_, err := git.PlainClone("./storage/liquid", false, &git.CloneOptions{
		// The intended use of a GitHub personal access token is in replace of your password
		// because access tokens can easily be revoked.
		// https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
		Auth: &http.BasicAuth{
			Username: "whatever", // yes, this can be anything except an empty string
			Password: "05303fba719627b0853c53b93a24de679938ef5f",
		},
		URL:      "https://github.com/Shopify/liquid.git",
		Progress: os.Stdout,
	})
	if err != nil {
		fmt.Println("boom:", err)
		return err
	}
	return nil
}
