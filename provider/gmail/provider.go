package gmail

import (
	"encoding/json"
	"fmt"
	"time"

	"agent-mail/model"
	"agent-mail/provider"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func init() {
	provider.RegisterProvider("gmail", NewProvider)
	provider.RegisterProviderFormInfo(provider.ProviderFormInfo{
		Type:  "gmail",
		Label: "Gmail",
		Fields: []provider.FieldDef{
			{Key: "client_id", Label: "Client ID", Type: "text", Section: "auth_data", Required: true},
			{Key: "client_secret", Label: "Client Secret", Type: "password", Section: "auth_data", Required: true},
			{Key: "access_token", Label: "Access Token", Type: "text", Section: "auth_data", Required: true},
			{Key: "refresh_token", Label: "Refresh Token", Type: "text", Section: "auth_data", Required: true},
			{Key: "token_expiry", Label: "Token Expiry (RFC3339)", Type: "text", Section: "auth_data", Required: true},
		},
	})
}

type GmailAuthData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TokenExpiry  string `json:"token_expiry"`
}

func newGmailService(auth GmailAuthData) (*gmail.Service, error) {
	expiry, err := time.Parse(time.RFC3339, auth.TokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("parse token expiry: %w", err)
	}

	config := &oauth2.Config{
		ClientID:     auth.ClientID,
		ClientSecret: auth.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailReadonlyScope, gmail.GmailSendScope, gmail.GmailModifyScope},
	}

	token := &oauth2.Token{
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		Expiry:       expiry,
	}

	client := config.Client(oauth2.NoContext, token)

	svc, err := gmail.NewService(oauth2.NoContext, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}

	return svc, nil
}

func NewProvider(record model.MailboxRecord) (*provider.MailProvider, error) {
	var auth GmailAuthData
	if err := json.Unmarshal([]byte(record.AuthData), &auth); err != nil {
		return nil, fmt.Errorf("parse gmail auth data: %w", err)
	}

	svc, err := newGmailService(auth)
	if err != nil {
		return nil, fmt.Errorf("init gmail service: %w", err)
	}

	return &provider.MailProvider{
		Receiver: &GmailReceiver{srv: svc},
		Sender:   &GmailSender{srv: svc, auth: auth},
	}, nil
}
