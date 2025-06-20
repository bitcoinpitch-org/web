package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
)

// TwitterConfig holds the OAuth2 configuration for Twitter
var TwitterConfig *oauth2.Config

// TwitterUser represents the Twitter user data we get from the API
type TwitterUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// InitTwitterOAuth initializes the Twitter OAuth configuration
func InitTwitterOAuth() {
	TwitterConfig = &oauth2.Config{
		ClientID:     os.Getenv("TWITTER_API_KEY"),
		ClientSecret: os.Getenv("TWITTER_API_SECRET"),
		RedirectURL:  os.Getenv("TWITTER_REDIRECT_URL"),
		Scopes:       []string{"tweet.read", "users.read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://twitter.com/i/oauth2/authorize",
			TokenURL: "https://api.twitter.com/2/oauth2/token",
		},
	}
}

// GetTwitterAuthURL generates the Twitter OAuth authorization URL
func GetTwitterAuthURL(state string) string {
	if TwitterConfig == nil {
		InitTwitterOAuth()
	}
	return TwitterConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCodeForToken exchanges the authorization code for an access token
func ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	if TwitterConfig == nil {
		InitTwitterOAuth()
	}
	return TwitterConfig.Exchange(ctx, code)
}

// GetTwitterUserInfo retrieves user information from Twitter API
func GetTwitterUserInfo(ctx context.Context, token *oauth2.Token) (*TwitterUser, error) {
	if TwitterConfig == nil {
		InitTwitterOAuth()
	}

	client := TwitterConfig.Client(ctx, token)

	// Get user info from Twitter API v2
	resp, err := client.Get("https://api.twitter.com/2/users/me")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Twitter API error: %d", resp.StatusCode)
	}

	var response struct {
		Data TwitterUser `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &response.Data, nil
}

// ValidateTwitterConfig checks if required environment variables are set
func ValidateTwitterConfig() error {
	if os.Getenv("TWITTER_API_KEY") == "" {
		return fmt.Errorf("TWITTER_API_KEY environment variable is required")
	}
	if os.Getenv("TWITTER_API_SECRET") == "" {
		return fmt.Errorf("TWITTER_API_SECRET environment variable is required")
	}
	if os.Getenv("TWITTER_REDIRECT_URL") == "" {
		return fmt.Errorf("TWITTER_REDIRECT_URL environment variable is required")
	}
	return nil
}
