package imgstore

type Store interface {
	View

	Install(newimg *Image) (oldimg *Image, err error)
	// remove the bits
	Unlink(img *Image) (error)
	Uninstall(repo string, tag string) (uninstalled View, err error)
}

type View interface {
	ListRepos() ([]string, error)
	ListTags(repo string) ([]string, error)
	Get(repo, tag string) (current *Image, previous []*Image, error)

	// Would this do a pull?
	ByHash(hash string) (*Image, error)
	All() ([]*Image, error)

	RepoEquals(string) View
	TagEquals(string) View
	RepoLike(string) View
	TagLike(string) View
	TagGT(string) View
	TagLT(string) View
}
