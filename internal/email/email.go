package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

type Mailer struct {
	cfg models.SMTPConfig
}

func New(cfg models.SMTPConfig) *Mailer { return &Mailer{cfg: cfg} }

func (m *Mailer) send(to []string, subject, body string) error {
	if m.cfg.Host == "" || len(to) == 0 {
		return nil
	}
	msg := "From: " + m.cfg.From + "\r\n" +
		"To: " + strings.Join(to, ",") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		body
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	if m.cfg.UseTLS {
		tlsCfg := &tls.Config{ServerName: m.cfg.Host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Auth(auth); err != nil {
			return err
		}
		if err := c.Mail(m.cfg.From); err != nil {
			return err
		}
		for _, t := range to {
			if err := c.Rcpt(t); err != nil {
				return err
			}
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		defer w.Close()
		_, err = fmt.Fprint(w, msg)
		return err
	}
	return smtp.SendMail(addr, auth, m.cfg.From, to, []byte(msg))
}

func render(tmpl string, data any) string {
	t := template.Must(template.New("").Parse(tmpl))
	var buf bytes.Buffer
	_ = t.Execute(&buf, data)
	return buf.String()
}

func (m *Mailer) DepSubmitted(dep models.Dependency, recipients []string) error {
	body := render(`<p>New dependency <b>{{.Name}}</b> submitted by {{.SubmittedBy}} is awaiting review.</p>`, dep)
	return m.send(recipients, "[next-deps-srv] New dependency pending review: "+dep.Name, body)
}

func (m *Mailer) DepApproved(dep models.Dependency, to string) error {
	body := render(`<p>Your dependency <b>{{.Name}}</b> has been approved and will be built shortly.</p>`, dep)
	return m.send([]string{to}, "[next-deps-srv] Dependency approved: "+dep.Name, body)
}

func (m *Mailer) DepRejected(dep models.Dependency, to, note string) error {
	type d struct {
		models.Dependency
		Note string
	}
	body := render(`<p>Your dependency <b>{{.Name}}</b> was rejected.</p><p>Reason: {{.Note}}</p>`, d{dep, note})
	return m.send([]string{to}, "[next-deps-srv] Dependency rejected: "+dep.Name, body)
}

func (m *Mailer) BuildDone(dep models.Dependency, job models.BuildJob, recipients []string) error {
	type d struct {
		Dep models.Dependency
		Job models.BuildJob
	}
	body := render(`<p>Build for <b>{{.Dep.Name}}</b> completed. {{.Job.FilesIndexed}} files indexed.</p>`, d{dep, job})
	return m.send(recipients, "[next-deps-srv] Build done: "+dep.Name, body)
}

func (m *Mailer) BuildFailed(dep models.Dependency, job models.BuildJob, recipients []string) error {
	type d struct {
		Dep models.Dependency
		Job models.BuildJob
	}
	body := render(`<p>Build for <b>{{.Dep.Name}}</b> failed.</p><p>Error: {{.Job.Error}}</p>`, d{dep, job})
	return m.send(recipients, "[next-deps-srv] Build failed: "+dep.Name, body)
}

func (m *Mailer) UserRegistered(user models.User, adminEmails []string) error {
	body := render(`<p>New user <b>{{.Username}}</b> ({{.Email}}) registered.</p>`, user)
	return m.send(adminEmails, "[next-deps-srv] New user registered: "+user.Username, body)
}

func (m *Mailer) RoleChanged(user models.User, newRoles []string) error {
	type d struct {
		models.User
		NewRoles string
	}
	body := render(`<p>Your roles have been updated to: <b>{{.NewRoles}}</b>.</p>`, d{user, strings.Join(newRoles, ", ")})
	return m.send([]string{user.Email}, "[next-deps-srv] Roles updated", body)
}
