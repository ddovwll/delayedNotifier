package notifier

import (
	"net/smtp"

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

func (n *EmailNotifier) Notify(recipient, message string) error {
	e := email.NewEmail()
	e.From = n.from
	e.To = []string{recipient}
	e.Subject = n.subject
	e.Text = []byte(message)

	err := e.Send(n.smtpAddr,
		smtp.PlainAuth("", n.username, n.password, n.host))
	if err != nil {
		return err
	}

	return nil
}
