package gmailc

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"log"
	"os"
)

func NewService(ctx context.Context) (*gmail.Service, error) {
	// 1. Load credentials.json from Google Cloud
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		return nil, fmt.Errorf("cannot read credentials.json: %w", err)
	}

	// 2. Configure OAuth with required scopes
	config, err := google.ConfigFromJSON(b,
		gmail.GmailModifyScope,
		gmail.GmailLabelsScope,
		gmail.GmailComposeScope,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot parse credentials.json: %w", err)
	}

	// 3. Get token from file or start OAuth flow
	tok, err := tokenFromFile("token.json")
	if err != nil {
		log.Println("token.json not found – starting OAuth flow")
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		saveToken("token.json", tok)
	}

	// 4. Create Gmail Service
	srv, err := gmail.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, tok)))
	if err != nil {
		return nil, fmt.Errorf("cannot create gmail service: %w", err)
	}

	return srv, nil
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Println("1) Copy this URL and open it in your browser:")
	fmt.Println(authURL)
	fmt.Println("\n2) Sign in and accept the permissions.")
	fmt.Print("3) Paste the authorization code here: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("cannot read auth code: %w", err)
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("cannot exchange code for token: %w", err)
	}
	return tok, nil
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tok oauth2.Token
	if err := json.NewDecoder(f).Decode(&tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func saveToken(path string, tok *oauth2.Token) {
	f, err := os.Create(path)
	if err != nil {
		log.Printf("cannot save token: %v", err)
		return
	}
	defer f.Close()

	_ = json.NewEncoder(f).Encode(tok)
	log.Printf("token saved to %s\n", path)
}
