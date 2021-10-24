package clii

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

const tokenPlace = "token.json"

func SaveToken(tok *oauth2.Token) error {
	f, err := os.OpenFile(tokenPlace, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open token file: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(tok); err != nil {
		return fmt.Errorf("failed to encode token: %w", err)
	}
	return nil
}

func UseToken() (*oauth2.Token, error) {
	f, err := os.Open(tokenPlace)
	if err != nil {
		return nil, fmt.Errorf("failed to open token file: %w", err)
	}
	defer f.Close()

	var tok oauth2.Token
	dec := json.NewDecoder(f)
	if err := dec.Decode(&tok); err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}
	return &tok, nil
}
