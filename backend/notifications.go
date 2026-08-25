package backend

import (
	"bytes"
	"errors"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/wtsi-hgi/softpack/db"
)

const (
	VersionConflict        = "The following packages have unmet dependencies"
	BuildStatusTemplate    = templatePath + "buildStatus.tmpl"
	PackageRequestTemplate = templatePath + "packageRequest.tmpl"
)

var ErrMissingAddress = errors.New("to and/or from email address not specified")

type statusTmpl struct {
	Success    bool
	Username   string
	Path       string
	BuildError bool
}

func (s *Server) GenerateEmailContents(tmpl string, data any) ([]byte, error) {
	temp, err := template.ParseGlob(tmpl)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if err := temp.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *Server) SendBuildStatusEmail(env *db.Environment) error {
	if env.Requester == "" || s.config.AdminAddr == "" {
		return ErrMissingAddress
	}

	success := false
	buildError := false

	if env.Status == db.Concretised {
		success = true
	} else if !strings.Contains(env.FailureReason, VersionConflict) {
		buildError = true
	}

	msg, err := s.GenerateEmailContents(BuildStatusTemplate, statusTmpl{
		Success:    success,
		Username:   env.Requester,
		Path:       env.Path,
		BuildError: buildError,
	})
	if err != nil {
		return err
	}

	return smtp.SendMail( //nolint:gosec
		s.config.SMTP,
		nil,
		s.config.AdminAddr,
		[]string{env.Requester + s.config.EmailDomain},
		msg,
	)
}

func (s *Server) SendPackageRequestEmail(req db.RecipeRequest) error {
	msg, err := s.GenerateEmailContents(PackageRequestTemplate, req)
	if err != nil {
		return err
	}

	return smtp.SendMail(s.config.SMTP, nil, req.Requester+s.config.EmailDomain, []string{s.config.AdminAddr}, msg)
}
