package backend

import (
	"strconv"
	"testing"

	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/backend/templates"
	"github.com/wtsi-hgi/softpack/db"
)

func TestGenerateStatusEmail(t *testing.T) {
	backend := newBackend(t, "")

	contents, err := backend.GenerateEmailContents(templates.BuildStatusTemplate, statusTmpl{
		Success:  true,
		Username: "capybara",             //nolint:goconst
		Path:     "/path/to/environment", //nolint:goconst
	})

	assert.NoError(t, err)
	assert.Equal(t, "Subject: Your SoftPack environment is ready!\r\n\r\n"+
		"Hi capybara,\n\nYour environment, /path/to/environment has built successfully.\n"+
		"SoftPack Team", string(contents))

	contents, err = backend.GenerateEmailContents(templates.BuildStatusTemplate, statusTmpl{
		Success:    false,
		Username:   "capybara",
		Path:       "/path/to/environment",
		BuildError: false,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Subject: Your SoftPack environment failed to build.\r\n\r\n"+
		"Hi capybara,\n\nYour environment, /path/to/environment has failed to build.\n"+
		"The error was a version conflict. Try relaxing which versions you've specified.\n"+
		"\nSoftPack Team", string(contents))

	contents, err = backend.GenerateEmailContents(templates.BuildStatusTemplate, statusTmpl{
		Success:    false,
		Username:   "capybara",
		Path:       "/path/to/environment",
		BuildError: true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Subject: Your SoftPack environment failed to build.\r\n\r\n"+
		"Hi capybara,\n\nYour environment, /path/to/environment has failed to build.\n"+
		"The error was a build error. Please contact your SoftPack administrator.\n"+
		"\nSoftPack Team", string(contents))
}

func TestGenerateRequestEmail(t *testing.T) {
	backend := newBackend(t, "")

	contents, err := backend.GenerateEmailContents(templates.PackageRequestTemplate, db.RecipeRequest{
		Name:      "capy",
		Version:   "2",
		URL:       "/path/to/capy",
		Details:   "desc",
		Requester: "sky",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Subject: SoftPack Recipe Request: capy\r\n\r\n"+
		"User: sky\nPackage: capy\nVersion: 2\nURL: /path/to/capy\nDescription: desc", string(contents))
}

func TestSendEmail(t *testing.T) {
	backend := newBackend(t, "")
	s, smtp := newMockSMTP(t)

	backend.config.SMTP = smtp
	backend.config.AdminAddr = "admin@mock.com"
	backend.config.EmailDomain = "@mocker.com"

	env := db.Environment{
		Path:          "/path/to/env",
		Requester:     "requester",
		Status:        db.Failed,
		FailureReason: "The following packages have unmet dependencies",
	}

	err := backend.SendBuildStatusEmail(&env)
	assert.NoError(t, err)

	mail := s.MessagesAndPurge()[0]
	m := mail.MsgRequest()

	assert.Contains(t, m, "Subject: Your SoftPack environment failed to build.")
	assert.Contains(t, m, "Hi requester,")
	assert.Contains(t, m, "Your environment, /path/to/env has failed to build.")
	assert.Contains(t, m, "The error was a version conflict.")
	assert.Contains(t, m, "Try relaxing which versions you've specified.")
	assert.Contains(t, m, "SoftPack Team")

	assert.Contains(t, mail.MailfromRequest(), "admin@mock.com")
	rcpt := mail.RcpttoRequestResponse()
	assert.Equal(t, "RCPT TO:<requester@mocker.com>", rcpt[0][0])
	assert.Equal(t, "250 Received", rcpt[0][1])

	req := db.RecipeRequest{
		Name:      "pkg",
		Version:   "2",
		URL:       "/path/to/pkg",
		Details:   "desc",
		Requester: "requester",
	}

	err = backend.SendPackageRequestEmail(req)
	assert.NoError(t, err)

	mail = s.MessagesAndPurge()[0]
	m = mail.MsgRequest()

	assert.Contains(t, m, "Subject: SoftPack Recipe Request: pkg")
	assert.Contains(t, m, "User: requester")
	assert.Contains(t, m, "Package: pkg")
	assert.Contains(t, m, "Version: 2")
	assert.Contains(t, m, "URL: /path/to/pkg")
	assert.Contains(t, m, "Description: desc")

	assert.Contains(t, mail.MailfromRequest(), "requester@mocker.com")
	rcpt = mail.RcpttoRequestResponse()
	assert.Equal(t, "RCPT TO:<admin@mock.com>", rcpt[0][0])
	assert.Equal(t, "250 Received", rcpt[0][1])
}

func newMockSMTP(t *testing.T) (*smtpmock.Server, string) {
	t.Helper()

	addr := "127.0.0.1"

	s := smtpmock.New(smtpmock.ConfigurationAttr{
		LogServerActivity: true,
		HostAddress:       addr,
	})

	err := s.Start()
	assert.NoError(t, err)

	smtp := addr + ":" + strconv.Itoa(s.PortNumber())

	return s, smtp
}
