package repos

type Repo interface {
	URL() string
	Name() string
	CloneTo(dstDir string) error
	Index() error
	OnCommitHook() error
}
