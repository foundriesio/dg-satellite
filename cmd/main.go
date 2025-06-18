package main

import (
	"github.com/alexflint/go-arg"
)

type CommonArgs struct {
	DataDir string `arg:"required" help:"Directory to store data"`
}

func main() {
	args := CommonArgs{}
	p := arg.MustParse(&args)

	switch {
	default:
		p.Fail("missing required subcommand")
	}
}
