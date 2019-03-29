package repos

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	//git "gopkg.in/src-d/go-git.v4"
	//"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"log"
	"os/exec"
	//"os"
	"strings"
)

type GithubRepo struct {
	r            *github.Repository
	cloneAbsPath string
}

func LoadAllGHRepos() ([]*github.Repository, error) {
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
			log.Println(err)
			continue
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
		fmt.Println("appended:", len(repos), " total:", len(allRepos))
		break
	}

	return allRepos, nil
}

func NewGithubRepo(r *github.Repository, absoluteStorageDirPath string) (*GithubRepo, error) {
	if r.FullName == nil {
		return nil, errors.New("nil fullname " + r.String())
	}
	if r.CloneURL == nil {
		return nil, errors.New("nil cloneURL " + r.String())
	}

	directory := strings.TrimSuffix(absoluteStorageDirPath, "/") + "/" + *r.FullName
	fmt.Println("cloneing:", *r.SSHURL)
	/*_, err := git.PlainClone(directory, false, &git.CloneOptions{
		// The intended use of a GitHub personal access token is in replace of your password
		// because access tokens can easily be revoked.
		// https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
		Auth: &http.BasicAuth{
			Username: "whatever", // yes, this can be anything except an empty string
			Password: "05303fba719627b0853c53b93a24de679938ef5f",
		},
		URL: *r.CloneURL,
		//Progress: os.Stdout,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating new repo failed")
	}*/
	cmd := exec.Command("git", "clone", *r.SSHURL, absoluteStorageDirPath)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return &GithubRepo{
		cloneAbsPath: directory,
		r:            r,
	}, nil
}

func (g *GithubRepo) Name() string {
	return *g.r.FullName
}
