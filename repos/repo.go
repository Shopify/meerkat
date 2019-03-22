package repos

import (
	"strings"
)

type Repo interface {
	WebURL() string
	Name() string
	CloneToDisk() error
	DiskPath() string
}

type repo struct {
	webURL   string
	diskPath string
}

func NewRepo(webURL, diskPath string) Repo {
	return &repo{
		webURL:   webURL,
		diskPath: diskPath,
	}
}

func (r *repo) WebURL() string {
	return r.webURL
}
func (r *repo) Name() string {
	idx := strings.LastIndex(r.webURL, "/")
	if idx < 0 {
		return r.webURL
	}
	ru := []rune(r.webURL)
	return string(ru[idx:])
}
func (*repo) CloneToDisk() error {
	return nil
}
func (r *repo) DiskPath() string {
	return r.diskPath
}
