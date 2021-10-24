package drive

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const MimeShortcut = "application/vnd.google-apps.shortcut"
const MimeFolder = "application/vnd.google-apps.folder"

// escapeListQuery https://developers.google.com/drive/api/v2/ref-search-terms
func escapeListQuery(s string) string {
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

func findFileIDByPath(srv *drive.Service, path string) (string, error) {
	names := strings.Split(strings.Trim(path, "/"), "/")

	if len(names) == 1 && names[0] == "" {
		return "root", nil
	}

	currentID := "root"
	currentPath := ""
	for i, name := range names {
		currentPath += "/" + name // for log

		if name == "" {
			return "", fmt.Errorf("invalid input, empty directory or file name: %s", currentPath)
		}

		isFolder := i != len(names)-1
		query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", escapeListQuery(name), currentID)
		if isFolder {
			query += fmt.Sprintf(" and (mimeType = '%s' or mimeType = '%s')", MimeFolder, MimeShortcut)
		}

		files, err := srv.Files.List().Fields("files(id, name, mimeType, shortcutDetails)").Q(query).Do()
		if err != nil {
			return "", fmt.Errorf("failed to search with query(%s): %w", query, err)
		}
		if len(files.Files) == 0 {
			return "", fmt.Errorf("path %s is missing: %w", currentPath, fs.ErrNotExist)
		}
		if len(files.Files) != 1 {
			// something go wrong
			return "", fmt.Errorf("path(%s) is not singular: %d files have same path", currentPath, len(files.Files))
		}

		file := files.Files[0]
		currentID = file.Id
		if file.MimeType == MimeShortcut {
			if file.ShortcutDetails == nil {
				return "", errors.New("something went wrong: shortcut but file.ShortcutDetails is nil")
			}
			currentID = file.ShortcutDetails.TargetId
			// memo: not shortcut files are already checked by query
			if isFolder && file.ShortcutDetails.TargetMimeType != MimeFolder {
				return "", fmt.Errorf("path (%s) is expected as directory, but not", currentPath)
			}
		}
	}
	return currentID, nil
}

// FindFileByPath find files in MyDrive
//
// Resolves shortcut.
// If applicable file doesn't exist, returns error wraps fs.ErrNotExist.
func FindFileByPath(ctx context.Context, hc *http.Client, path string) (*drive.File, error) {
	srv, err := drive.NewService(ctx, option.WithHTTPClient(hc))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize drive service: %w", err)
	}

	fileID, err := findFileIDByPath(srv, path)
	if err != nil {
		return nil, err
	}

	f, err := srv.Files.Get(fileID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file (id=%s): %w", fileID, err)
	}
	return f, nil
}
