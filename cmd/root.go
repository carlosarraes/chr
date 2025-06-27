package cmd

import (
	"context"
	"os"
)

func Execute(ctx context.Context) error {
	return ExecuteCLI(os.Args[1:])
}