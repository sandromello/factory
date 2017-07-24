package conf

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

type Config struct {
	DockerAddr     string
	RegistryOrg    string
	RegistryURL    string
	PushToRegistry bool
	CloneInfo      CloneInfo
	GitSecret      GitSecret
	RegistrySecret RegistrySecret
}

type CloneInfo struct {
	URL       string
	Path      string
	Ref       string
	Commit    string
	ImageName string
	ImageTag  string
	Overwrite bool
}

type GitAuthType string

const (
	GitOauthType     GitAuthType = "git-oauth"
	GitBasicAuthType GitAuthType = "git-basic"
	GitNoAuth        GitAuthType = ""
)

type Registry map[string]RegistrySecret

type RegistrySecret struct {
	RegUsername string `json:"username"`
	RegPassword string `json:"password"`
	Email       string `json:"email"`
	Auth        string `json:"auth"` // username:password in base64
	// RegBase64   string
}

type GitSecret struct {
	Username   string
	Password   string
	OauthToken string
}

func (c Config) GitAuthType() GitAuthType {
	if len(c.GitSecret.OauthToken) > 0 {
		return GitOauthType
	}
	if len(c.GitSecret.Username) > 0 {
		return GitBasicAuthType
	}
	return GitNoAuth
}

func (c Config) CloneOptions() (*git.CloneOptions, error) {
	opts := &git.CloneOptions{
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(c.CloneInfo.Ref),
	}
	gitURL, err := url.Parse(c.CloneInfo.URL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing clone url, %v", err)
	}
	opts.URL = gitURL.String()
	switch c.GitAuthType() {
	case GitOauthType:
		gitURL.User = url.UserPassword(c.GitSecret.OauthToken, "x-oauth-basic")
		opts.URL = gitURL.String()
	case GitBasicAuthType:
		opts.Auth = http.NewBasicAuth(c.GitSecret.Username, c.GitSecret.Password)
	}
	return opts, nil
}

func (c Config) GetImageTag() string {
	tag := c.CloneInfo.Commit
	if len(tag) == 0 {
		tag = c.CloneInfo.ImageTag
		if len(tag) == 0 {
			tag = "v1"
		}
	}
	return tag
}

// RegistryAuth returns a base64 representation of the secret
func (c Config) RegistryAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf(`{"username":"%s","password":"%s"}`,
			c.RegistrySecret.RegUsername,
			c.RegistrySecret.RegPassword,
		),
	))
}
