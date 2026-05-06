package mail

import (
	"crypto/tls"
	"fmt"

	"github.com/go-mail/mail/v2"
	"github.com/sirupsen/logrus"
	"github.com/st0mb1e/bank-service-go/config"
)

type Mailer struct {
	cfg *config.SMTPConfig
	log *logrus.Logger
}

func NewMailer(cfg *config.SMTPConfig, log *logrus.Logger) *Mailer {
	return &Mailer{cfg: cfg, log: log}
}

func (m *Mailer) SendPaymentNotification(toEmail string, amount string) error {
	if !m.cfg.Enabled {
		return nil
	}
	msg := mail.NewMessage()
	msg.SetHeader("From", m.cfg.From)
	msg.SetHeader("To", toEmail)
	msg.SetHeader("Subject", "Платёж")
	msg.SetBody("text/html", fmt.Sprintf(
		`<p>Списание/платёж на сумму <strong>%s RUB</strong></p>`, amount))

	d := mail.NewDialer(m.cfg.Host, m.cfg.Port, m.cfg.User, m.cfg.Password)
	d.TLSConfig = &tls.Config{ServerName: m.cfg.Host}
	if err := d.DialAndSend(msg); err != nil {
		m.log.Errorf("smtp: %v", err)
		return fmt.Errorf("email: %w", err)
	}
	return nil
}
