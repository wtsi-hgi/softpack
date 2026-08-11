package backend

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
)

func TestGenerateStatusEmail(t *testing.T) {
	backend := newBackend(t)
	if backend.config.SMTP == "" {
		t.Skip("EMAIL_TEST_SMTP not set")
	}

	contents, err := backend.GenerateEmailContents(BuildStatusTemplate, statusTmpl{
		Success:  true,
		Username: "capybara",             //nolint:goconst
		Path:     "/path/to/environment", //nolint:goconst
	})

	assert.NoError(t, err)
	assert.Equal(t, "Subject: Your SoftPack environment is ready!\r\n\r\n"+
		"Hi capy,\n\nYour environment, /path/to/environment has built successfully.\n"+
		"SoftPack Team", string(contents))

	contents, err = backend.GenerateEmailContents(BuildStatusTemplate, statusTmpl{
		Success:    false,
		Username:   "capybara",
		Path:       "/path/to/environment",
		BuildError: false,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Subject: Your SoftPack environment failed to build.\r\n\r\n"+
		"Hi capy,\n\nYour environment, /path/to/environment has failed to build.\n"+
		"The error was a version conflict. Try relaxing which versions you've specified.\n"+
		"\nSoftPack Team", string(contents))

	contents, err = backend.GenerateEmailContents(BuildStatusTemplate, statusTmpl{
		Success:    false,
		Username:   "capybara",
		Path:       "/path/to/environment",
		BuildError: true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Subject: Your SoftPack environment failed to build.\r\n\r\n"+
		"Hi capy,\n\nYour environment, /path/to/environment has failed to build.\n"+
		"The error was a build error. Please contact your SoftPack administrator.\n"+
		"\nSoftPack Team", string(contents))
}

func TestGenerateRequestEmail(t *testing.T) {
	backend := newBackend(t)
	if backend.config.SMTP == "" {
		t.Skip("EMAIL_TEST_SMTP not set")
	}

	contents, err := backend.GenerateEmailContents(PackageRequestTemplate, db.RecipeRequest{
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
	if os.Getenv("RUN_EMAIL_TEST") != "true" {
		t.Skip("RUN_EMAIL_TEST is not set to true")
	}

	from := os.Getenv("EMAIL_TEST_FROM")
	to := os.Getenv("EMAIL_TEST_TO")
	domain := os.Getenv("EMAIL_TEST_DOMAIN")

	if from == "" { //nolint:gocritic,nestif
		t.Skip("EMAIL_TEST_FROM is empty")
	} else if to == "" {
		t.Skip("EMAIL_TEST_TO is empty")
	} else if domain == "" {
		t.Skip("EMAIL_TEST_DOMAIN is empty")
	}

	backend := newBackend(t)
	if backend.config.SMTP == "" {
		t.Skip("EMAIL_TEST_SMTP not set")
	}

	env := db.Environment{
		Path:          "/path/to/env",
		Requester:     to,
		Status:        db.Failed,
		FailureReason: "The following packages have unmet dependencies",
	}

	err := backend.SendBuildStatusEmail(&env)
	assert.NoError(t, err)
}
