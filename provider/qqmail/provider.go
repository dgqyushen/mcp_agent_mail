package qqmail

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"agent-mail/model"
	"agent-mail/provider"
	"github.com/emersion/go-imap/v2/imapclient"
)

func init() {
	provider.RegisterProvider("qqmail", NewProvider)
	provider.RegisterProviderFormInfo(provider.ProviderFormInfo{
		Type:  "qqmail",
		Label: "QQMail",
		Fields: []provider.FieldDef{
			{Key: "username", Label: "邮箱地址", Type: "text", Section: "auth_data", Required: true},
			{Key: "password", Label: "授权码", Type: "password", Section: "auth_data", Required: true},
		},
	})
}

type QQmailAuthData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewProvider(record model.MailboxRecord) (*provider.MailProvider, error) {
	var auth QQmailAuthData
	if err := json.Unmarshal([]byte(record.AuthData), &auth); err != nil {
		return nil, fmt.Errorf("invalid qqmail auth data: %w", err)
	}

	recv := &QQmailReceiver{auth: auth}
	send := &QQmailSender{auth: auth}
	return &provider.MailProvider{Receiver: recv, Sender: send}, nil
}

func newIMAPClient(auth QQmailAuthData) (*imapclient.Client, error) {
	client, err := imapclient.DialTLS("imap.qq.com:993", &imapclient.Options{
		TLSConfig: &tls.Config{ServerName: "imap.qq.com"},
	})
	if err != nil {
		return nil, fmt.Errorf("imap dial: %w", err)
	}
	if err := client.Login(auth.Username, auth.Password).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("imap login: %w", err)
	}
	return client, nil
}
