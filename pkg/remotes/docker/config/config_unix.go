package config

import (
	"crypto/x509"
	"path/filepath"
)

func hostPaths(root, host string) (hosts []string) {
	ch := hostDirectory(host)
	if ch != host {
		hosts = append(hosts, filepath.Join(root, ch))
	}

	hosts = append(hosts,
		filepath.Join(root, host),
		filepath.Join(root, "_default"),
	)

	return
}

func rootSystemPool() (*x509.CertPool, error) {
	return x509.SystemCertPool()
}
