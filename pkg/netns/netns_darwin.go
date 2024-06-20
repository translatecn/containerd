package netns

// NetNS holds network namespace.
type NetNS struct {
	path string
}

func NewNetNS(baseDir string) (*NetNS, error) {
	return nil, nil
}

func (n *NetNS) GetPath() string {
	return ""
}

func (n *NetNS) Remove() error {
	return nil
}
