package tfbin

import "fmt"

type ProviderMeta struct {
	Host      string
	Namespace string
	Name      string
	OS        string
	Arch      string
	Version   string
}

func (poa ProviderMeta) Equals(p2 ProviderMeta) bool {
	if poa.Arch != p2.Arch {
		return false
	}
	if poa.OS != p2.OS {
		return false
	}
	if poa.Name != p2.Name {
		return false
	}
	if poa.Namespace != p2.Namespace {
		return false
	}
	if poa.Host != p2.Host {
		return false
	}
	if poa.Version != p2.Version {
		return false
	}
	return true
}

func (pm ProviderMeta) FindMatch(options []ProviderMeta) (ProviderMeta, error) {
	for _, opt := range options {
		if opt.Equals(pm) {
			return opt, nil
		}
	}
	return ProviderMeta{}, &NoMatchError{ProviderMeta: pm}
}

func (pm ProviderMeta) String() string {
	return fmt.Sprintf("host=%s, namespace=%s, name=%s, os=%s, arch=%s, version=%s",
		pm.Host, pm.Namespace, pm.Name, pm.OS, pm.Arch, pm.Version)
}

type NoMatchError struct {
	ProviderMeta ProviderMeta
}

func (e *NoMatchError) Error() string {
	return "No results for query: " + e.ProviderMeta.String()
}
