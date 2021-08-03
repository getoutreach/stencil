package github

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v34/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	OutreachOrganization = "getoutreach"
	GithubFilePath       = "/.outreach/github.token"
)

type Github struct {
	*github.Client
	OrgName string
}

func NewGithub(orgName, tokenFilePath string) (*Github, error) {
	gh := new(Github)
	gh.OrgName = orgName
	d, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(d, tokenFilePath)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		//	Please go to https://github.com/settings/tokens and create a personal access token
		// with repo: and user: access and copy it to ~/.outreach/github.token
		//
		// Please see here for details:
		//  https://outreach-io.atlassian.net/wiki/spaces/EN/pages/784041501/Creating+a+github+personal+access+token

		message := `Failed to get user personal token from "%s"`
		return nil, errors.Wrapf(err, message, filePath)
	}

	ctx := context.Background()
	token := &oauth2.Token{AccessToken: strings.TrimSpace(string(data))}
	gh.Client = github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)))
	return gh, nil
}
