package mailer

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Mailer struct {
	SmtpServer   string      `yaml:"mail_smtp_server"`
	MailUser     string      `yaml:"mail_auth_user"`
	MailPassword string      `yaml:"mail_auth_password"`
	Headers      *MailHeader `yaml:"general,inline"`
	Logger       *log.Logger
	body         string
}

type MailHeader struct {
	From        string   `yaml:"mail_address_from"`
	To          []string `yaml:"mailing_list"`
	Subject     string
	contentType string
}

// return array of formatted strings which look like:
// "'%unitName%' %unitType% got error: %errorString%"
func (m *Mailer) getErrorString(unitName string, unitType string, errors []*error) []string {
	var errorString []string

	level.Debug(*m.Logger).Log("msg", "create array of strings with errors")

	for _, e := range errors {
		errorString = append(errorString, fmt.Sprintf("'%s' %s got error: %s\n", unitName, unitType, *e))
	}

	return errorString
}

func (m *Mailer) getErrorsHtml(errorStrings []string) string {
	var htmlBody string

	level.Debug(*m.Logger).Log("msg", "create full html with errors")

	for _, s := range errorStrings {
		htmlBody += fmt.Sprintf("		<br>%s\n", s)
	}

	return fmt.Sprintf(`
<html>
	<head></head>
	<body>
%s
	</body>
</html>`, htmlBody)
}

func (m *Mailer) SendHtmlEmail(unitName string, unitType string, errors []*error) error {
	var err error

	level.Debug(*m.Logger).Log("msg", "send HTML email with errors")
	serviceErrorString := m.getErrorString(unitName, unitType, errors)
	m.body = m.getErrorsHtml(serviceErrorString)

	m.Headers.contentType = "text/html; charset=utf-8"
	smtpAuth := smtp.PlainAuth("", m.MailUser, m.MailPassword, strings.Split(m.SmtpServer, ":")[0])

	// Prepare message as RFC-822 formatted
	msg := []byte(fmt.Sprintf(`Subject: %s
Content-Type: %s

%s`, m.Headers.Subject, m.Headers.contentType, m.body))
	level.Debug(*m.Logger).Log("msg", "send email", "server", m.SmtpServer,
		"from", m.Headers.From, "to", fmt.Sprintf("%+v", m.Headers.To),
		"value", string(msg))

	if err = smtp.SendMail(m.SmtpServer, smtpAuth, m.Headers.From, m.Headers.To, msg); err != nil {
		fmt.Println("EEE:", err)
		level.Error(*m.Logger).Log("msg", "got error when try to send email", "error", err.Error())
	}

	return err
}
