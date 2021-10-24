package drive

import (
	"context"
	"fmt"
	"net/http"

	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type copyOption struct {
	ParentID string
}

func (c copyOption) toFile() drive.File {
	f := drive.File{}
	if c.ParentID != "" {
		f.Parents = append(f.Parents, c.ParentID)
	}

	return f
}

type CopyOption func(d *drive.Service, o *copyOption) error

func CopyToDirectoryID(parentDirectoryID string) CopyOption {
	return func(_ *drive.Service, o *copyOption) error {
		o.ParentID = parentDirectoryID
		return nil
	}
}

func Copy(ctx context.Context, hc *http.Client, sourceFileID string, options ...CopyOption) error {
	srv, err := drive.NewService(ctx, option.WithHTTPClient(hc))
	if err != nil {
		return fmt.Errorf("failed to initialize drive service: %w", err)
	}

	var opt copyOption
	for _, apply := range options {
		if err := apply(srv, &opt); err != nil {
			return fmt.Errorf("failed to apply option: %w", err)
		}
	}

	optFile := opt.toFile()
	if _, err := srv.Files.Copy(sourceFileID, &optFile).SupportsAllDrives(true).Do(); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
