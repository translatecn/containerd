package main

import (
	b64 "encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"

	encconfig "github.com/containers/ocicrypt/config"
	cryptUtils "github.com/containers/ocicrypt/utils"
)

// getDecryptionKeys reads the keys from the given directory
func getDecryptionKeys(keysPath string) (encconfig.CryptoConfig, error) {
	var cc encconfig.CryptoConfig

	base64Keys := make([]string, 0)

	walkFn := func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Handle symlinks
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			return errors.New("Symbolic links not supported in decryption keys paths")
		}

		privateKey, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// TODO - Remove the need to covert to base64. The ocicrypt library
		// should provide a method to directly process the private keys
		sEnc := b64.StdEncoding.EncodeToString(privateKey)
		base64Keys = append(base64Keys, sEnc)

		return nil
	}

	err := filepath.Walk(keysPath, walkFn)
	if err != nil {
		return cc, err
	}

	sortedDc, err := cryptUtils.SortDecryptionKeys(strings.Join(base64Keys, ","))
	if err != nil {
		return cc, err
	}

	cc = encconfig.InitDecryption(sortedDc)

	return cc, nil
}

func combineDecryptionConfigs(dc1, dc2 *encconfig.DecryptConfig) *encconfig.DecryptConfig {
	cc1 := encconfig.CryptoConfig{
		DecryptConfig: dc1,
	}
	cc2 := encconfig.CryptoConfig{
		DecryptConfig: dc2,
	}

	cc := encconfig.CombineCryptoConfigs([]encconfig.CryptoConfig{cc1, cc2})
	return cc.DecryptConfig
}
