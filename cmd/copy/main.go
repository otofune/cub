package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/otofune/cub/drive"
	"github.com/otofune/cub/internal/clii"
	"github.com/otofune/cub/internal/mainenv"
)

func realMain(ctx context.Context, env mainenv.Env) error {
	targetFolPath := os.Args[1]
	targetIDs := os.Args[2:]

	if len(targetIDs) == 0 {
		return errors.New("please give id")
	}

	tok, err := clii.UseToken()
	if err != nil {
		return fmt.Errorf("please authorize w/ cmd/auth-local before using cli: %w", err)
	}
	conf := env.GoogleOAuth2Config()
	hc := conf.Client(ctx, tok)

	dirFile, err := drive.FindFileByPath(ctx, hc, targetFolPath)
	if err != nil {
		return fmt.Errorf("failed to find target folder: %w", err)
	}
	if dirFile.MimeType != drive.MimeFolder {
		return errors.New("given path found, but not a folder")
	}

	for _, id := range targetIDs {
		if err := drive.Copy(ctx, hc, id, drive.CopyToDirectoryID(dirFile.Id)); err != nil {
			return fmt.Errorf("failed to copy %s: %w", id, err)
		}
	}

	return nil
}

func main() {
	clii.Run(realMain)
}
