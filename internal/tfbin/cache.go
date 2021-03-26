package tfbin

import (
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Cache struct {
	fs   afero.Fs
	base string
}

const DefaultCacheDirName string = ".tf-cache"

// NewDefaultCache returns a cache with basePath
// set to $CWD/.tf-cache
func NewDefaultCache(afs afero.Fs) (*Cache, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return NewCache(afs, path.Join(cwd, DefaultCacheDirName))
}

func NewCache(afs afero.Fs, basePath string) (*Cache, error) {
	abs, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}
	exists, err := afero.DirExists(afs, abs)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = afs.MkdirAll(abs, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return &Cache{fs: afs, base: basePath}, nil
}

func (c *Cache) ProviderMetaWriter(pm ProviderMeta, filename string) (io.WriteCloser, error) {
	dir := path.Join(c.base, pm.Host, pm.Namespace, pm.Name, pm.Version, pm.OS, pm.Arch)
	err := c.fs.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	writePath := path.Join(dir, filename)
	return c.fs.Create(writePath)
}

func (c *Cache) ProviderMetaReader(pm ProviderMeta) (io.ReadCloser, string, error) {
	dir := path.Join(c.base, pm.Host, pm.Namespace, pm.Name, pm.Version, pm.OS, pm.Arch)
	files, err := afero.ReadDir(c.fs, dir)
	if err != nil {
		return nil, "", err
	}
	if len(files) > 1 {
		return nil, "", errors.New("Unexpected duplicate files in cache at %s" + dir)
	}
	f := files[0]
	fh, err := c.fs.Open(path.Join(dir, f.Name()))
	return fh, f.Name(), err
}

func (c *Cache) Search(query ProviderMeta) (ProviderMeta, error) {
	f := &optionFinder{base: c.base, found: make([]ProviderMeta, 0)}
	err := afero.Walk(c.fs, c.base, f.Walk)
	if err != nil {
		return ProviderMeta{}, err
	}
	return query.FindMatch(f.found)
}

type optionFinder struct {
	base  string
	found []ProviderMeta
}

const (
	hostPosition      int = 0
	namespacePosition int = 1
	namePosition      int = 2
	versionPosition   int = 3
	osPosition        int = 4
	archPosition      int = 5
)

func (f *optionFinder) Walk(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}
	dir, _ := filepath.Split(path)
	rel, err := filepath.Rel(f.base, dir)
	if err != nil {
		return err
	}

	dirs := strings.Split(rel, string(os.PathSeparator))
	if len(dirs) == 5 {
		dirs = append([]string{""}, dirs...)
	}
	pm := ProviderMeta{
		Host:      dirs[hostPosition],
		Namespace: dirs[namespacePosition],
		Name:      dirs[namePosition],
		Version:   dirs[versionPosition],
		OS:        dirs[osPosition],
		Arch:      dirs[archPosition],
	}
	f.found = append(f.found, pm)
	return nil
}
