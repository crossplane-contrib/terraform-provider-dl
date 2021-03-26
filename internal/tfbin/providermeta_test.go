package tfbin

import (
	"fmt"
	"testing"

	"github.com/Masterminds/semver"
)

func TestChooserPicksHighest(t *testing.T) {
	p := ProviderMeta{
		Name:      "test",
		Namespace: "upbound",
		OS:        "TempleOS",
		Arch:      "cthulu64",
		Host:      "localhost",
	}
	version := "1.23.4"
	expected := newTestPM(p.Host, p.Namespace, version, p.Name, p.OS, p.Arch)
	options := []ProviderMeta{
		newTestPM(p.Host, p.Namespace, "1.21.0", p.Name, p.OS, p.Arch),
		newTestPM(p.Host, p.Namespace, "1.22.0", p.Name, p.OS, p.Arch),
		newTestPM(p.Host, p.Namespace, "1.23.0", p.Name, p.OS, p.Arch),
		expected,
		newTestPM(p.Host, p.Namespace, "1.23.4", "nope", p.OS, p.Arch),
		newTestPM(p.Host, p.Namespace, "1.23.4", p.Name, "nopeOs", p.Arch),
		newTestPM(p.Host, p.Namespace, "1.23.4", p.Name, p.OS, "noThulu64"),
		newTestPM(p.Host, p.Namespace, "1.24.0", p.Name, p.OS, p.Arch),
	}
	err := checkExpected(expected, options)
	if err != nil {
		t.Error(err)
	}
}

func TestChooserNoMatch(t *testing.T) {
	p := ProviderMeta{
		Name:      "test",
		Namespace: "upbound",
		OS:        "TempleOS",
		Arch:      "cthulu64",
		Host:      "localhost",
	}
	version := "1.23.4"
	expected := newTestPM(p.Host, p.Namespace, version, p.Name, p.OS, p.Arch)
	options := []ProviderMeta{
		newTestPM(p.Host, p.Namespace, "1.23.4", "nope", p.OS, p.Arch),
		newTestPM(p.Host, p.Namespace, "1.23.4", p.Name, "nopeOs", p.Arch),
		newTestPM(p.Host, p.Namespace, "1.23.4", p.Name, p.OS, "noThulu64"),
	}
	err := checkExpected(expected, options)
	if err == nil {
		t.Errorf("Got a matching option where no valid options were provided")
	}
}

func checkExpected(expected ProviderMeta, options []ProviderMeta) error {
	observed, err := expected.FindMatch(options)
	if err != nil {
		return fmt.Errorf("Unexpected error from ProviderQuery.TopRanked = %s", err)
	}

	if !expected.Equals(observed) {
		return fmt.Errorf("FindMatch chose an unexpected ProviderMeta. Expected=%s, Actual=%s", expected, observed)
	}
	return nil
}

type testOption struct {
	host          string
	versionString string
	namespace     string
	name          string
	os            string
	arch          string
}

func (to *testOption) Version() *semver.Version {
	v, _ := semver.NewVersion(to.versionString)
	return v
}

func (to *testOption) ProviderMeta() ProviderMeta {
	return ProviderMeta{Name: to.name, Namespace: to.namespace, OS: to.os, Arch: to.arch}
}

func newTestPM(host, namespace, version, provider, os, arch string) ProviderMeta {
	return ProviderMeta{Host: host, Version: version, Namespace: namespace, Name: provider, OS: os, Arch: arch}
}
