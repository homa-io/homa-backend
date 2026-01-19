package email

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
)

// XOAuth2Client implements SASL XOAUTH2 authentication for IMAP.
type XOAuth2Client struct {
	Username    string
	AccessToken string
}

// NewXOAuth2Client creates a new XOAUTH2 SASL client.
func NewXOAuth2Client(username, accessToken string) sasl.Client {
	return &XOAuth2Client{
		Username:    username,
		AccessToken: accessToken,
	}
}

// Start begins the XOAUTH2 authentication.
func (c *XOAuth2Client) Start() (string, []byte, error) {
	// XOAUTH2 format: "user=" + email + "^Aauth=Bearer " + accessToken + "^A^A"
	// Where ^A is ASCII 0x01 (SOH)
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.Username, c.AccessToken)
	return "XOAUTH2", []byte(authString), nil
}

// Next handles the next challenge (XOAUTH2 doesn't have challenges).
func (c *XOAuth2Client) Next(challenge []byte) ([]byte, error) {
	// If we get a challenge, it means authentication failed
	// The challenge contains the error message
	return nil, fmt.Errorf("authentication failed: %s", string(challenge))
}

// XOAuth2Auth implements smtp.Auth for XOAUTH2.
type XOAuth2Auth struct {
	Username    string
	AccessToken string
}

// NewXOAuth2Auth creates a new XOAUTH2 auth for SMTP.
func NewXOAuth2Auth(username, accessToken string) *XOAuth2Auth {
	return &XOAuth2Auth{
		Username:    username,
		AccessToken: accessToken,
	}
}

// Start begins the XOAUTH2 authentication for SMTP.
func (a *XOAuth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.Username, a.AccessToken)
	return "XOAUTH2", []byte(authString), nil
}

// Next handles SMTP authentication challenges.
func (a *XOAuth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("unexpected challenge: %s", string(fromServer))
	}
	return nil, nil
}

// OAuthTokenResponse represents the response from token refresh.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// RefreshGmailAccessToken refreshes a Gmail OAuth2 access token.
func RefreshGmailAccessToken(clientID, clientSecret, refreshToken string) (string, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	return refreshOAuth2Token(tokenURL, data)
}

// RefreshOutlookAccessToken refreshes an Outlook OAuth2 access token.
func RefreshOutlookAccessToken(clientID, clientSecret, tenantID, refreshToken string) (string, error) {
	// Use "common" if tenant not specified
	if tenantID == "" {
		tenantID = "common"
	}
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"scope":         {"https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/SMTP.Send offline_access"},
	}

	return refreshOAuth2Token(tokenURL, data)
}

// refreshOAuth2Token performs the OAuth2 token refresh request.
func refreshOAuth2Token(tokenURL string, data url.Values) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// GetGmailAuthURL returns the OAuth2 authorization URL for Gmail.
func GetGmailAuthURL(clientID, redirectURI, state string) string {
	params := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"https://mail.google.com/ https://www.googleapis.com/auth/gmail.send"},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
		"state":         {state},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// GetOutlookAuthURL returns the OAuth2 authorization URL for Outlook.
func GetOutlookAuthURL(clientID, tenantID, redirectURI, state string) string {
	if tenantID == "" {
		tenantID = "common"
	}
	params := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/SMTP.Send offline_access"},
		"state":         {state},
	}
	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s", tenantID, params.Encode())
}

// ExchangeGmailCode exchanges an authorization code for tokens.
func ExchangeGmailCode(clientID, clientSecret, code, redirectURI string) (*OAuthTokenResponse, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	return exchangeCode(tokenURL, data)
}

// ExchangeOutlookCode exchanges an authorization code for tokens.
func ExchangeOutlookCode(clientID, clientSecret, tenantID, code, redirectURI string) (*OAuthTokenResponse, error) {
	if tenantID == "" {
		tenantID = "common"
	}
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
		"scope":         {"https://outlook.office.com/IMAP.AccessAsUser.All https://outlook.office.com/SMTP.Send offline_access"},
	}

	return exchangeCode(tokenURL, data)
}

// exchangeCode performs the OAuth2 code exchange.
func exchangeCode(tokenURL string, data url.Values) (*OAuthTokenResponse, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return &tokenResp, nil
}

// EncodeXOAuth2 encodes the XOAUTH2 string for SASL.
func EncodeXOAuth2(username, accessToken string) string {
	authString := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", username, accessToken)
	return base64.StdEncoding.EncodeToString([]byte(authString))
}
