package notifier

import (
	"context"
	"os"

	"github.com/jordan-wright/email"
)

type EmailNotifier struct {
	from     string
	subject  string
	smtpAddr string
	username string
	password string
	host     string
}

func NewEmailNotifier() *EmailNotifier {
	from := os.Getenv("MAIL_FROM")
	subject := os.Getenv("MAIL_SUBJECT")
	smtpAddr := os.Getenv("MAIL_SMTP_ADDR")
	username := os.Getenv("MAIL_USERNAME")
	password := os.Getenv("MAIL_PASSWORD")
	host := os.Getenv("MAIL_HOST")
	return &EmailNotifier{
		from:     from,
		subject:  subject,
		smtpAddr: smtpAddr,
		username: username,
		password: password,
		host:     host,
	}
}

func (n *EmailNotifier) Notify(_ context.Context, recipient, message string) error {
	e := email.NewEmail()
	e.From = n.from
	e.To = []string{recipient}
	e.Subject = n.subject
	e.Text = []byte(message)

	err := e.Send(n.smtpAddr,
		nil)
	if err != nil {
		return err
	}

	return nil
}
