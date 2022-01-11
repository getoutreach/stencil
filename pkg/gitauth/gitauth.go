// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description.

// Package gitauth provides helpers for setting up auth
package gitauth

import (
	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/go-git/go-git/v5"
	"github.com/sirupsen/logrus"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"

	giturls "github.com/whilp/git-urls"
)

type protocol string

const (
	protocolSSH   protocol = "SSH"
	protocolHTTPS protocol = "HTTPS"
)

// ensureURLIsValidForProtocol ensures that a provided gitUrl is valid for the given
// protocol by parsing it into a URL and then returning a valid URL for the provided
// protocol
func ensureURLIsValidForProtocol(opts *git.CloneOptions, expectedProtocol protocol) error {
	u, err := giturls.Parse(opts.URL)
	if err != nil {
		return err
	}

	switch expectedProtocol {
	case protocolSSH:
		u.Scheme = "ssh"
	case protocolHTTPS:
		u.Scheme = "https"
	}

	opts.URL = u.String()

	return nil
}

// configureAccessTokenAuth sets up Github access token authentication
func configureAccessTokenAuth(token cfg.SecretData, opts *git.CloneOptions) error {
	opts.Auth = &githttp.BasicAuth{
		Username: "x-access-token",
		Password: string(token),
	}

	return ensureURLIsValidForProtocol(opts, protocolHTTPS)
}

// ConfigureAuth configures the provided git.CloneOptions to be authenticated for
// Github repository clones
func ConfigureAuth(accessToken cfg.SecretData, opts *git.CloneOptions,
	log logrus.FieldLogger) error {
	// Don't setup auth if no auth token is set
	if accessToken == "" {
		return nil
	}

	return configureAccessTokenAuth(accessToken, opts)
}
