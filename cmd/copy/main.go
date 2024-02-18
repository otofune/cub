package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/otofune/cub/drive"
	"github.com/otofune/cub/internal/clii"
	"github.com/otofune/cub/internal/mainenv"
	"golang.org/x/exp/maps"

	gdrive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const stateFile = "./state.json"

func readState(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var s []string
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return nil, err
	}

	rs := make(map[string]struct{})
	for _, v := range s {
		rs[v] = struct{}{}
	}
	return rs, nil
}

func writeState(path string, state map[string]struct{}) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(maps.Keys(state))
}

func realMain(ctx context.Context, env mainenv.Env) error {
	state, err := readState(stateFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		state = make(map[string]struct{})
	}
	defer func() {
		if err := writeState(stateFile, state); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save state: %v\n", err)
		}
	}()

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	for i, id := range maps.Keys(targetIDs) {
		if _, ok := state[id]; ok {
			fmt.Printf("[SKIP] %d / %d (%s)\n", i, len(targetIDs), id)
			continue
		}
		fmt.Printf("[COPYING] %d / %d (%s)\n", i, len(targetIDs), id)
		if err := drive.Copy(ctx, hc, id, drive.CopyToDirectoryID(dirFile.Id)); err != nil {
			return fmt.Errorf("failed to copy %s: %w", id, err)
		}
		state[id] = struct{}{}

		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "exiting by signal\n")
			return nil
		default:
		}
	}

	return nil
}

func main() {
	clii.Run(realMain)
}
