package clii

import (
	"context"
	"fmt"
	"os"

	"github.com/otofune/cub/internal/mainenv"
)

func Run(realMain func(c context.Context, e mainenv.Env) error) {
	ctx := context.Background()
	var env mainenv.Env
	if err := env.Process(); err != nil {
		fmt.Printf("failed to start because env missing: %v", err)
		os.Exit(1)
	}

	if err := realMain(ctx, env); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
