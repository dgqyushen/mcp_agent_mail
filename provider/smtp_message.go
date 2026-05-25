package provider

import (
	"fmt"
	"strings"
)

func BuildSMTPMessage(fromName, fromEmail, toMail, toName, subject, content string, isHTML bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, fromEmail))
	b.WriteString(fmt.Sprintf("To: %s <%s>\r\n", toName, toMail))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	if isHTML {
		b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	} else {
		b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	}
	b.WriteString("\r\n")
	b.WriteString(content)
	return b.String()
}
