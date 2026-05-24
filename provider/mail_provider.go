package provider

import "errors"

type MailProvider struct {
	Receiver MailReceiver
	Sender   MailSender
}

var ErrCapNotSupported = errors.New("capability not supported by this provider")
