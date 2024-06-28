package main

import (
	"demo/pkg/typeurl"
	"fmt"
	"io"
	"os"

	"demo/others/imgcrypt"
	"demo/others/imgcrypt/images/encryption"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/urfave/cli"
)

var (
	Usage = "ctd-decoder is used as a call-out from containerd content stream plugins"
)

func main() {
	app := cli.NewApp()
	app.Name = "ctd-decoder"
	app.Usage = Usage
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "decryption-keys-path",
			Usage: "Path to load decryption keys from. (optional)",
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	if err := decrypt(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
		return err
	}
	return nil
}

func decrypt(ctx *cli.Context) error {
	payload, err := getPayload()
	if err != nil {
		return err
	}

	decCc := &payload.DecryptConfig

	// TODO: If decryption key path is set, get additional keys to augment payload keys
	if ctx.GlobalIsSet("decryption-keys-path") {
		keyPathCc, err := getDecryptionKeys(ctx.GlobalString("decryption-keys-path"))
		if err != nil {
			return fmt.Errorf("unable to get decryption keys in provided key path: %w", err)
		}
		decCc = combineDecryptionConfigs(keyPathCc.DecryptConfig, &payload.DecryptConfig)
	}

	_, r, _, err := encryption.DecryptLayer(decCc, os.Stdin, payload.Descriptor, false)
	if err != nil {
		return fmt.Errorf("call to DecryptLayer failed: %w", err)
	}

	for {
		_, err := io.CopyN(os.Stdout, r, 10*1024)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("could not copy data: %w", err)
		}
	}
	return nil
}

func getPayload() (*imgcrypt.Payload, error) {
	data, err := readPayload()
	if err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	var anything types.Any
	if err := proto.Unmarshal(data, &anything); err != nil {
		return nil, fmt.Errorf("could not proto.Unmarshal() decrypt data: %w", err)
	}
	v, err := typeurl.UnmarshalAny(&anything)
	if err != nil {
		return nil, fmt.Errorf("could not UnmarshalAny() the decrypt data: %w", err)
	}
	l, ok := v.(*imgcrypt.Payload)
	if !ok {
		return nil, fmt.Errorf("unknown payload type %s", anything.TypeUrl)
	}
	return l, nil
}

const payloadFD = 3

func readPayload() ([]byte, error) {
	f := os.NewFile(payloadFD, "configFd")
	defer f.Close()
	return io.ReadAll(f)
}
