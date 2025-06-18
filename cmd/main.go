// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexflint/go-arg"
)

type CommonArgs struct {
	DataDir string `arg:"required" help:"Directory to store data"`

	Csr *CsrCmd `arg:"subcommand:create-csr" help:"Create a TLS certificate signing request for this server"`
}

func (c CommonArgs) CertsDir() string {
	return filepath.Join(c.DataDir, "certs")
}

func (c CommonArgs) MkDirs() error {
	for _, x := range []string{c.DataDir, c.CertsDir()} {
		if err := os.Mkdir(x, 0o740); err != nil {
			return fmt.Errorf("unable to create data directory(%s): %w", x, err)
		}
	}
	return nil
}

func main() {
	args := CommonArgs{}
	p := arg.MustParse(&args)

	switch {
	case args.Csr != nil:
		if err := args.Csr.Run(args); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		}
	default:
		p.Fail("missing required subcommand")
	}
}
