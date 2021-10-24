package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/otofune/cub/internal/clii"
	"github.com/otofune/cub/internal/mainenv"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

func realMain(ctx context.Context, env mainenv.Env) error {
	conf := env.GoogleOAuth2Config()
	conf.Scopes = append(conf.Scopes, drive.DriveScope)

	state := time.Now().String()
	url := conf.AuthCodeURL(state, oauth2.AccessTypeOnline) // web = online

	fmt.Printf("Open %s \n", url)

	authReqChan := make(chan struct {
		Code  string
		State string
	})
	authServer := http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", env.LISTEN_PORT),
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			state := r.URL.Query().Get("state")
			if code != "" && state != "" {
				authReqChan <- struct {
					Code  string
					State string
				}{
					Code:  code,
					State: state,
				}
			}
			rw.Write([]byte("OK"))
		}),
	}

	go func() {
		if err := authServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalln("failed to setup authServer:", err)
		}
	}()
	defer func() {
		sctx, _ := context.WithTimeout(ctx, time.Second*10)
		if err := authServer.Shutdown(sctx); err != nil {
			log.Fatalln("failed to shutdown auth server:", err)
		}
	}()

	for req := range authReqChan {
		if req.State != state {
			fmt.Println("mismatch state. please retry")
			continue
		}
		ectx, ectxcan := context.WithTimeout(ctx, time.Second*10)
		defer ectxcan()

		tok, err := conf.Exchange(ectx, req.Code)
		if err != nil {
			return fmt.Errorf("failed to exchange token: %w", err)
		}

		if err := clii.SaveToken(tok); err != nil {
			return err
		}

		break
	}

	fmt.Println("ALL OK, token saved to token.json")

	return nil
}

func main() {
	clii.Run(realMain)
}
