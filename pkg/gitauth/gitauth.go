// Copyright 2021 Outreach Corporation. All Rights Reserved.

// Description: See package description.

// Package gitauth provides helpers for setting up auth
package gitauth

import (
	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/sshhelper"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
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

func configureSSHAuth(sshKey string, opts *git.CloneOptions, log logrus.FieldLogger) error {
	a := sshhelper.GetSSHAgent()
	// if user gave us a key, use that
	if sshKey != "" {
		err := sshhelper.AddKeyToAgent(sshKey, a, log)
		if err != nil {
			return errors.Wrap(err, "failed to load specified ssh-key")
		}

		opts.Auth = sshhelper.NewExistingSSHAgentCallback(a)
		return nil
	}

	sshKey, err := sshhelper.LoadDefaultKey("github.com", a, log)
	if err == nil {
		log.WithField("key", sshKey).Info("using IdentityFile for host github.com")
		opts.Auth = sshhelper.NewExistingSSHAgentCallback(a)
		return nil
	}

	log.WithError(err).Warn("failed to load github ssh key, falling back to ssh-agent")
	log.Warn("Falling back to ssh-agent authentication")

	return ensureURLIsValidForProtocol(opts, protocolSSH)
}

func configureAccessTokenAuth(token cfg.SecretData, opts *git.CloneOptions) error {
	opts.Auth = &githttp.BasicAuth{
		Username: "x-access-token",
		Password: string(token),
	}

	return ensureURLIsValidForProtocol(opts, protocolHTTPS)
}

// ConfigureAuth configures the provided git.CloneOptions to be authenticated for
// Github repository clones.
func ConfigureAuth(sshKeyPath string, accessToken cfg.SecretData, opts *git.CloneOptions, log logrus.FieldLogger) error {
	if accessToken != "" {
		return configureAccessTokenAuth(accessToken, opts)
	}

	// attempt to use ssh-key, or load the default key
	return configureSSHAuth(sshKeyPath, opts, log)
}
