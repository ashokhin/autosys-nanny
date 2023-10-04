package mailer

import (
	"bytes"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Mailer struct {
	SmtpServer    string      `yaml:"mail_smtp_server"`
	MailUser      string      `yaml:"mail_auth_user"`
	MailPassword  string      `yaml:"mail_auth_password"`
	SubjectPrefix string      `yaml:"mail_subject_prefix"`
	Headers       *MailHeader `yaml:"general,inline"`
	passwordRunes []rune      // most safe storage for password in memory
	Logger        *log.Logger
}

func (m *Mailer) String() string {
	return fmt.Sprintf("%+v", *m)
}

type MailHeader struct {
	From        string   `yaml:"mail_address_from"`
	To          []string `yaml:"mailing_list"`
	ContentType string   `yaml:"mail_content_type"`
	Subject     string
}

const MAIL_DEFAULT_CONTENT_TYPE string = `text/plain; charset="utf-8"`

func (h *MailHeader) String() string {
	return fmt.Sprintf("%+v", *h)
}

func (m *Mailer) SafeStorePassword() {
	m.passwordRunes = []rune(m.MailPassword)
	m.MailPassword = ""
}

func (m *Mailer) CheckSettings() error {
	if m.Headers == nil {
		m.Headers = new(MailHeader)
	}

	switch {
	case len(m.SmtpServer) == 0:
		return &ErrBadMailSettings{"mail settings don't have 'mail_smtp_server' value"}
	case !strings.Contains(m.SmtpServer, ":"):
		return &ErrBadMailSettings{"mail settings don't have smtp server port number in 'mail_smtp_server' field"}
	case len(m.Headers.From) == 0:
		hostname, _ := os.Hostname()
		hostnameArray := strings.Split(hostname, ".")
		if len(hostnameArray) >= 2 {
			m.Headers.From = fmt.Sprintf("%s@%s",
				// get first part of FQDN. For example "server01" from "server01.sub.example.com"
				hostnameArray[0],
				// get 1st level domain part of FQDN. For example "example.com" from "server01.sub.example.com"
				hostnameArray[len(hostnameArray)-2:])

		} else {
			m.Headers.From = hostnameArray[0]
		}
		level.Warn(*m.Logger).Log("msg", "mail settings don't have 'mail_address_from' field. Use default value",
			"value", m.Headers.From)

	}

	return nil
}

// return array of formatted strings which look like:
// "Host '%hostname%' got error: %errorString%"
func (m *Mailer) getErrorString(errorsStringArray []*error) []string {
	var errorString []string

	level.Debug(*m.Logger).Log("msg", "create array of strings with errors")

	hostname, _ := os.Hostname()

	for _, e := range errorsStringArray {
		errorString = append(errorString, fmt.Sprintf("Host '%s' got error: %s\n", hostname, *e))
	}

	return errorString
}

func (m *Mailer) getErrorsBody(errorsArray []*error) string {
	var mailBody string

	errorsStringArray := m.getErrorString(errorsArray)

	switch {
	case strings.Contains(m.Headers.ContentType, "text/plain"):
		mailBody = strings.Join(errorsStringArray, "\n")
	case strings.Contains(m.Headers.ContentType, "text/html"):
		mailBody = m.getErrorsHtml(errorsStringArray)
	}

	return mailBody
}

func (m *Mailer) getErrorsHtml(errorsStringArray []string) string {
	var htmlBody string

	level.Debug(*m.Logger).Log("msg", "create full html with errors")

	for _, s := range errorsStringArray {
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

func (m *Mailer) buildEmail(mailBodyString string) []byte {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\n", m.Headers.From))
	buf.WriteString(fmt.Sprintf("To: %s\n", strings.Join(m.Headers.To, "; ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\n", m.Headers.Subject))
	buf.WriteString(fmt.Sprintf("Content-Type: %s\n", m.Headers.ContentType))
	buf.WriteString(fmt.Sprintf("\n%s", mailBodyString))

	return buf.Bytes()

}

func (m *Mailer) SendHtmlEmail(errors []*error) error {
	var err error
	var mailBodyString string
	var smtpAuth smtp.Auth

	level.Debug(*m.Logger).Log("msg", "send html email with errors")

	if len(m.Headers.ContentType) == 0 {
		m.Headers.ContentType = MAIL_DEFAULT_CONTENT_TYPE
	}

	if len(m.MailUser) > 0 && m.passwordRunes != nil {
		smtpAuth = smtp.PlainAuth("", m.MailUser, string(m.passwordRunes), strings.Split(m.SmtpServer, ":")[0])
	}

	mailBodyString = m.getErrorsBody(errors)

	// Prepare message as RFC-822 formatted
	messageBytes := m.buildEmail(mailBodyString)

	level.Debug(*m.Logger).Log("msg", "send email", "server", m.SmtpServer,
		"from", m.Headers.From, "to", fmt.Sprintf("%+v", m.Headers.To),
		"value", string(messageBytes))

	if err = smtp.SendMail(m.SmtpServer, smtpAuth, m.Headers.From, m.Headers.To, messageBytes); err != nil {
		level.Error(*m.Logger).Log("msg", "got error when try to send email", "error", err.Error())
	}

	return err
}
