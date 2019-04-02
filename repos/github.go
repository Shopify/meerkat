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
	"os"
	"os/exec"
	"strings"
)

const GHAccessToken = "DUMMY_TOKEN"

type GithubRepo struct {
	r            *github.Repository
	cloneAbsPath string
}

func LoadAllGHRepos() ([]*github.Repository, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: GHAccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// list all repositories for the authenticated user
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
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
	url := "https://" + GHAccessToken + "@" + strings.TrimPrefix(*r.GitURL, "git://")
	//fmt.Println("Cloning:", url)
	cmd := exec.Command("git", "clone", "--depth=1", url, directory)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	os.RemoveAll(directory + "/.git")

	return &GithubRepo{
		cloneAbsPath: directory,
		r:            r,
	}, nil
}

func (g *GithubRepo) Name() string {
	return *g.r.FullName
}

func (g *GithubRepo) DiskPath() string {
	return g.cloneAbsPath
}
