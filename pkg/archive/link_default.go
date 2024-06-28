package archive

import "os"

func link(oldname, newname string) error {
	return os.Link(oldname, newname)
}
