package main

import (
	"fmt"
	"github.com/crossplane-contrib/terraform-provider-dl/internal/tfbin"
	"github.com/spf13/afero"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/crossplane-contrib/terraform-provider-gen/pkg/provider"
)

var CLI struct {
	ProviderName string `help:"Official Terraform provider name; the name that would be used in a tf provider block."`
	Version      string `help:"Semantic version constraint for the desired provider. See github.com/Masterminds/semver for syntax"`
	OS           string `help:"Target operating system for provider binary."`
	Arch         string `help:"Target system architecture for provider binary."`
	Output       string `help:"The binary will be written to a subdirectory ensuring unique provider+version+arch within the --output location."`
	Config       string `help:"Path to a terraform-provider-gen config file."`
}

func main() {
	var err error
	ctx := kong.Parse(&CLI)

	if CLI.Config != "" {
		cfg, err := provider.ConfigFromFile(CLI.Config)
		if err != nil {
			ctx.FatalIfErrorf(err)
		}
		CLI.ProviderName = cfg.Name
		CLI.Version = cfg.ProviderVersion
	}

	if CLI.OS == "" {
		CLI.OS = runtime.GOOS
	}
	if CLI.Arch == "" {
		CLI.Arch = runtime.GOARCH
	}

	pm := tfbin.ProviderMeta{
		OS:      CLI.OS,
		Arch:    CLI.Arch,
		Version: CLI.Version,
	}
	nameParts := strings.Split(CLI.ProviderName, "/")
	switch len(nameParts) {
	case 1:
		pm.Namespace = "hashicorp"
		pm.Name = CLI.ProviderName
	case 2:
		pm.Name, pm.Namespace = nameParts[0], nameParts[1]
	default:
		ctx.FatalIfErrorf(fmt.Errorf("Expecting only one slash in provider name -- namespace/name. got: %s", CLI.ProviderName))
	}

	if CLI.Output == "" {
		CLI.Output, err = os.Getwd()
		if err != nil {
			ctx.FatalIfErrorf(err)
		}
	}
	CLI.Output, err = filepath.Abs(CLI.Output)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}

	afs := afero.NewOsFs()
	exists, err := afero.DirExists(afs, CLI.Output)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
	if !exists {
		err = afs.MkdirAll(CLI.Output, os.ModePerm)
		if err != nil {
			ctx.FatalIfErrorf(err)
		}
	}

	cache, err := tfbin.NewDefaultCache(afs)
	reg := tfbin.NewTerraformRegistry(tfbin.WithCache(cache))
	match, err := reg.Search(pm)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}

	reader, filename, err := reg.ProviderMetaReader(match)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
	defer reader.Close()

	writePath := path.Join(CLI.Output, filename)
	writer, err := afs.Create(writePath)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
}
