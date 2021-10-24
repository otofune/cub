package mainenv

import (
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type Env struct {
	GOOGLE_CLIENT_ID     string `envconfig:"GOOGLE_CLIENT_ID" required:"true"`
	GOOGLE_CLIENT_SECRET string `envconfig:"GOOGLE_CLIENT_SECRET" required:"true"`
	GOOGLE_REDIRECT_URL  string `envconfig:"GOOGLE_REDIRECT_URL" required:"true"`
	COOKIE_JWE_KEY       string `envconfig:"COOKIE_KEY" required:"true"`
	LISTEN_PORT          int    `envconfig:"PORT" required:"true"`
}

func (e *Env) Process() error {
	return envconfig.Process("", e)
}

// GoogleOAuth2Config returns oauth2.Config from env
//
// oauth2.Config.Scopes will be always empty, you must set before use.
func (e Env) GoogleOAuth2Config() oauth2.Config {
	return oauth2.Config{
		ClientID:     e.GOOGLE_CLIENT_ID,
		ClientSecret: e.GOOGLE_CLIENT_SECRET,
		RedirectURL:  e.GOOGLE_REDIRECT_URL,
		Endpoint:     endpoints.Google,
	}
}
