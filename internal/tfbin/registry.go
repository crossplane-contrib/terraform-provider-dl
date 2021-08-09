package tfbin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/pkg/errors"
)

type TerraformRegistry struct {
	client *registry.Client
	cache  *Cache
}

func (tr *TerraformRegistry) tfProvider(target ProviderMeta) *regsrc.TerraformProvider {
	name := target.Name
	if target.Namespace != "" {
		name = fmt.Sprintf("%s/%s", target.Namespace, name)
	}
	return regsrc.NewTerraformProvider(name, target.OS, target.Arch)
}

func (tr *TerraformRegistry) Search(query ProviderMeta) (ProviderMeta, error) {
	var err error
	if tr.cache != nil {
		cachedResult, err := tr.cache.Search(query)
		if err == nil {
			return cachedResult, nil
		} else {
			// fall through to the registry search if not found in cache
			// but barf if there is something actually wrong with the cache
			var e *NoMatchError
			if !errors.As(err, &e) {
				return ProviderMeta{}, err
			}
		}
	}

	resp, err := tr.client.TerraformProviderVersions(tr.tfProvider(query))
	if err != nil {
		return ProviderMeta{}, err
	}

	opts := make([]ProviderMeta, 0)
	for _, tpv := range resp.Versions {
		for _, p := range tpv.Platforms {
			pm := ProviderMeta{
				Host:      query.Host,
				Namespace: query.Namespace,
				Name:      query.Name,
				OS:        p.OS,
				Arch:      p.Arch,
				Version:   tpv.Version,
			}
			opts = append(opts, pm)
		}
	}
	return query.FindMatch(opts)
}

func (tr *TerraformRegistry) ProviderMetaReader(pm ProviderMeta) (io.ReadCloser, string, error) {
	if tr.cache != nil {
		rc, fname, err := tr.cache.ProviderMetaReader(pm)
		if err == nil {
			return rc, fname, err
		} else {

		}
	}
	loc, err := tr.client.TerraformProviderLocation(tr.tfProvider(pm), pm.Version)
	if err != nil {
		return nil, "", err
	}
	zipURL := loc.DownloadURL // retrieving the actual zip
	resp, err := http.Get(zipURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("GET of %s had a non-200 result: %d", zipURL, resp.StatusCode)
	}

	// unpack the zip onto the local filesystem
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("Error reading body from request to %s: %v", zipURL, err)
	}
	zread, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	pluginPrefix := fmt.Sprintf("terraform-provider-%s", pm.Name)
	found := false
	for i := range zread.File {
		zfh := zread.File[i]
		if !strings.HasPrefix(zfh.Name, pluginPrefix) {
			continue
		}
		found = true

		fh, err := zfh.Open()
		if err != nil {
			return nil, "", err
		}
		if tr.cache == nil {
			return fh, zfh.Name, nil
		}
		defer fh.Close()
		wc, err := tr.cache.ProviderMetaWriter(pm, zfh.Name)
		if err != nil {
			return nil, "", errors.Wrap(err, "Error initializing local cache to store provider binary.")
		}
		defer wc.Close()
		_, err = io.Copy(wc, fh)
		if err != nil {
			return nil, "", errors.Wrap(err, "Error writing provider binary from registry to local cache.")
		}
	}

	if !found {
		return nil, "", fmt.Errorf("provider plugin binary not found with prefix: %s", pluginPrefix)
	}
	return tr.cache.ProviderMetaReader(pm)
}

type TerraformRegistryOption func(terraformRegistry *TerraformRegistry)

func NewTerraformRegistry(options ...TerraformRegistryOption) *TerraformRegistry {
	client := registry.NewClient(nil, nil)
	tr := &TerraformRegistry{client: client}
	for _, o := range options {
		o(tr)
	}
	return tr
}

func WithCache(c *Cache) TerraformRegistryOption {
	return func(tr *TerraformRegistry) {
		tr.cache = c
	}
}
