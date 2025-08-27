// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"

	"github.com/foundriesio/dg-satellite/context"
)

type CommonArgs struct {
	DataDir string `arg:"required" help:"Directory to store data"`

	Csr     *CsrCmd     `arg:"subcommand:create-csr" help:"Create a TLS certificate signing request for this server"`
	SignCsr *CsrSignCmd `arg:"subcommand:sign-csr" help:"Create the TLS certificate from the signing request"`
	Serve   *ServeCmd   `arg:"subcommand:serve" help:"Run the REST API and device-gateway services"`

	ctx context.Context
}

func main() {
	log, err := context.InitLogger("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
		return
	}

	args := CommonArgs{
		ctx: context.CtxWithLog(context.Background(), log),
	}
	p := arg.MustParse(&args)

	switch {
	case args.Csr != nil:
		err = args.Csr.Run(args)
	case args.SignCsr != nil:
		err = args.SignCsr.Run(args)
	case args.Serve != nil:
		err = args.Serve.Run(args)
	default:
		p.Fail("missing required subcommand")
	}
	if err != nil {
		log.Error("command failed", "error", err)
		os.Exit(1)
	}
}
