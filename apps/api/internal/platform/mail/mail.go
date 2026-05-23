package mail

import (
	"fmt"
	"net/smtp"
)

type Sender interface {
	Send(to, subject, body string) error
}

type SMTPSender struct {
	Host string
	Port string
	From string
	User string
	Pass string
}

func NewSMTPSender(host, port, from, user, pass string) *SMTPSender {
	return &SMTPSender{
		Host: host,
		Port: port,
		From: from,
		User: user,
		Pass: pass,
	}
}

func (s *SMTPSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		s.From, to, subject, body)

	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Pass, s.Host)
	}

	return smtp.SendMail(addr, auth, s.From, []string{to}, []byte(msg))
}