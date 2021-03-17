package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/alecthomas/kong"
	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/response"
)

var CLI struct {
	ProviderName string `help:"Official Terraform provider name; the name that would be used in a tf provider block."`
	Version string `help:"Semantic version constraint for the desired provider. See github.com/Masterminds/semver for syntax"`
	OS string `help:"Target operating system for provider binary."`
	Arch string `help:"Target system architecture for provider binary."`
	Output string `help:"The binary will be written to a subdirectory ensuring unique provider+version+arch within the --output location."`
}

func main() {
	ctx := kong.Parse(&CLI)
	// arg defaults
	if len(strings.Split(CLI.ProviderName, "/")) < 2 {
		CLI.ProviderName = "hashicorp/" + CLI.ProviderName
	}
	vc, err := semver.NewConstraint(CLI.Version)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
	if CLI.OS == "" {
		CLI.OS = runtime.GOOS
	}
	if CLI.Arch == "" {
		CLI.Arch = runtime.GOARCH
	}

	// provider simply holds metadata about the provider we want to search the registry for
	provider := regsrc.NewTerraformProvider(CLI.ProviderName, CLI.OS, CLI.Arch)
	client := registry.NewClient(nil, nil)

	// will fail if we can't find a version in the registry that matches the version constraint + arch + os
	v, err := BestVersion(client, provider, vc)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}

	if CLI.Output != "" {
		CLI.Output, err = filepath.Abs(CLI.Output)
		if err != nil {
			ctx.FatalIfErrorf(err)
		}
	} else {
		// we found a suitable match, so set up local path to save the provider binary
		dir, err := os.Getwd()
		if err != nil {
			ctx.FatalIfErrorf(err)
		}
		osArch := fmt.Sprintf("%s_%s", provider.OS, provider.Arch)
		CLI.Output = path.Join(dir, "tf-plugin", provider.RawHost.String(), provider.RawNamespace, provider.RawName, v, osArch)
	}
	err = os.MkdirAll(CLI.Output, os.ModePerm)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}

	// specific registry entry for this provider/version/arch/os, result contains final zip download url
	zipURL, err := VersionZipURL(client, provider, v)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}

	// retrieving the actual zip
	resp, err := http.Get(zipURL)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		ctx.FatalIfErrorf(fmt.Errorf("GET of %s had a non-200 result: %d", zipURL, resp.StatusCode))
	}

	// unpack the zip onto the local filesystem
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ctx.FatalIfErrorf(fmt.Errorf("Error reading body from request to %s: %v", zipURL, err))
	}
	zread, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	for _, zfh := range zread.File {
		localPath := path.Join(CLI.Output, zfh.Name)
		fh, err := os.Create(localPath)
		if err != nil {
			ctx.FatalIfErrorf(fmt.Errorf("Failed to create local filehandle at %s to save filename '%s' from %s, with err=%v", localPath, zfh.Name, zipURL, err))
		}
		zfhReader, err := zfh.Open()
		if err != nil {
			ctx.FatalIfErrorf(fmt.Errorf("Error reading filename %s from zip retrieved from %s, with err=%v", zfh.Name, zipURL, err))
		}
		defer fh.Close()
		defer zfhReader.Close()
		_, err = io.Copy(fh, zfhReader)
		if err != nil {
			ctx.FatalIfErrorf(fmt.Errorf("Error copying between zip archive file %s to local file %s from zip retrieved from %s, with err=%v", zfh.Name, localPath, zipURL, err))
		}

		fmt.Printf("\nWrote provider plugin to local path %s\nWhere terraform-provider-gen or a terraform-provider-runtime based provider require a plugin path argument or environment variable, use the path of the directory containing the binary, ie:\n%s\n", localPath, CLI.Output)
	}
}

func VersionZipURL(client *registry.Client, provider *regsrc.TerraformProvider, version string) (string, error) {
	loc, err := client.TerraformProviderLocation(provider, version)
	if err != nil {
		return "", err
	}
	return loc.DownloadURL, nil
}

func BestVersion(client *registry.Client, provider *regsrc.TerraformProvider, vc *semver.Constraints) (string, error) {
	resp, err := client.TerraformProviderVersions(provider)
	if err != nil {
		return "", err
	}
	vmap := mapForPlatform(resp, provider.OS, provider.Arch)
	smvs := make([]*semver.Version, 0)
	for sv, _ := range vmap {
		v, err := semver.NewVersion(sv)
		if err != nil {
			return "", err
		}
		smvs = append(smvs, v)
	}
	sort.Sort(semver.Collection(smvs))
	for i := len(smvs); i > 0; i-- {
		candidate := smvs[i-1]
		if vc.Check(candidate) {
			return candidate.String(), nil
		}
	}
	return "", fmt.Errorf("Could not find a suitable match for the specified combination of provider, os, arch and version")
}

func mapForPlatform(vr *response.TerraformProviderVersions, os, arch string) map[string]*response.TerraformProviderVersion {
	m := make(map[string]*response.TerraformProviderVersion)
	for _, v := range vr.Versions {
		if platformSupported(v, os, arch) {
			m[v.Version] = v
		}
	}
	return m
}

func platformSupported(version *response.TerraformProviderVersion, os, arch string) bool {
	for _, p := range version.Platforms {
		if p.OS == os && p.Arch == arch {
			return true
		}
	}
	return false
}
