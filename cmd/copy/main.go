package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/otofune/cub/drive"
	"github.com/otofune/cub/internal/clii"
	"github.com/otofune/cub/internal/mainenv"
	"golang.org/x/exp/maps"

	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func realMain(ctx context.Context, env mainenv.Env) error {
	targetFolPath := os.Args[1]
	if len(os.Args[2:]) == 0 {
		return errors.New("please give id")
	}

	targetIDs := make(map[string]struct{})
	for _, id := range os.Args[2:] {
		targetIDs[id] = struct{}{}
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

	// directory copy
	gs, err := gdrive.NewService(ctx, option.WithHTTPClient(hc))
	if err != nil {
		return err
	}
	var orQuery []string
	for id := range targetIDs {
		orQuery = append(orQuery, fmt.Sprintf("'%s' in parents", id))
	}
	query := strings.Join(orQuery, " or ")

	if query != "" {
		// FIXME: このクエリは or で繋げているがどうもこれだと複数ディレクトリ配下を列挙できないらしい（naze?）
		fmt.Println(query)
		var childrenFiles []*gdrive.File
		{
			var nextPageToken string
			for {
				lr, err := gs.Files.List().Fields("nextPageToken, files(id, name, mimeType, shortcutDetails, parents)").Q(query).PageSize(100).SupportsAllDrives(true).PageToken(nextPageToken).Do()
				if err != nil {
					return fmt.Errorf("failed to get childrens: %w", err)
				}
				childrenFiles = append(childrenFiles, lr.Files...)
				if lr.NextPageToken == "" {
					break
				}
				nextPageToken = lr.NextPageToken
			}
		}
		fmt.Println(len(childrenFiles))
		parents := make(map[string]struct{})
		for _, f := range childrenFiles {
			for _, p := range f.Parents {
				parents[p] = struct{}{}
			}
			if strings.HasPrefix(f.MimeType, "application/vnd.google-") {
				fmt.Printf("skip %s, id: %s (cannot copy it)\n", f.Name, f.Id)
				continue
			}
			targetIDs[f.Id] = struct{}{}
		}
		for p := range parents {
			delete(targetIDs, p)
		}
	}

	for i, id := range maps.Keys(targetIDs) {
		fmt.Printf("Copying %d / %d (%s)\n", i, len(targetIDs), id)
		if err := drive.Copy(ctx, hc, id, drive.CopyToDirectoryID(dirFile.Id)); err != nil {
			return fmt.Errorf("failed to copy %s: %w", id, err)
		}
	}

	return nil
}

func main() {
	clii.Run(realMain)
}
