package secrets

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
    "io/ioutils"
	"github.com/dollarshaveclub/acyl/pkg/config"

	"github.com/pkg/errors"
)

var vaultSecrets = map[string]string{
	"rootPath"                   : "/vault/secrets/",
	"awsAccessKeyIDid"           : "awsAccessKeyIDid",
	"awsSecretAccessKeyid"       : "awsSecretAccessKeyid",
	"githubHookSecretid"         : "githubHookSecretid",
	"githubTokenid"              : "githubTokenid",
	"githubAppID"                : "githubAppID",
	"githubAppPK"                : "githubAppPK",
	"githubAppHookSecret"        : "githubAppHookSecret",
	"githubOAuthInstID"          : "githubOAuthInstID",
	"githubOAuthClientID"        : "githubOAuthClientID",
	"githubOAuthClientSecret"    : "githubOAuthClientSecret",
	"githubOAuthCookieEncKey"    : "githubOAuthCookieEncKey",
	"githubOAuthCookieAuthKey"   : "githubOAuthCookieAuthKey",
	"githubOAuthUserTokenEncKey" : "githubOAuthUserTokenEncKey",
	"apiKeysid"                  : "apiKeysid",
	"slackTokenid"               : "slackTokenid",
	"tlsCertid"                  : "tlsCertid",
	"tlsKeyid"                   : "tlsKeyid",
	"dbURIid"                    : "dbURIid",
}

type IOReaderSecretsFetcher struct {
	r *ioutils.Reader
}

func NewIOReaderSecretsFetcher(r *io.Reader) *IOReaderSecretsFetcher {
	return &IOReaderSecretsFetcher{
		r: r,
	}
}

// PopulateAllSecrets populates all secrets into the respective config structs
func (iof *IOReaderSecretsFetcher) PopulateAllSecrets(aws *config.AWSCreds, gh *config.GithubConfig, slack *config.SlackConfig, srv *config.ServerConfig, pg *config.PGConfig) error {
	if err := iof.PopulateAWS(aws); err != nil {
		return errors.Wrap(err, "error getting AWS secrets")
	}
	if err := iof.PopulateGithub(gh); err != nil {
		return errors.Wrap(err, "error getting GitHub secrets")
	}
	if err := iof.PopulateSlack(slack); err != nil {
		return errors.Wrap(err, "error getting Slack secrets")
	}
	if err := iof.PopulateServer(srv); err != nil {
		return errors.Wrap(err, "error getting server secrets")
	}
	if err := iof.PopulatePG(pg); err != nil {
		return errors.Wrap(err, "error getting db secrets")
	}
	return nil
}

// PopulatePG populates postgres secrets into pg
func (psf *PVCSecretsFetcher) PopulatePG(pg *config.PGConfig) error {
	s, err := iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["dbURIid"])
	if err != nil {
		return errors.Wrap(err, "error getting DB URI")
	}
	pg.PostgresURI = string(s)
	return nil
}

// PopulateAWS populates AWS secrets into aws
func (psf *PVCSecretsFetcher) PopulateAWS(aws *config.AWSCreds) error {
	s, err := iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["awsAccessKeyIDid"])
	if err != nil {
		return errors.Wrap(err, "error getting AWS access key ID")
	}
	aws.AccessKeyID = string(s)
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["awsSecretAccessKeyid"])
	if err != nil {
		return errors.Wrap(err, "error getting AWS secret access key")
	}
	aws.SecretAccessKey = string(s)
	return nil
}

// PopulateGithub populates Github secrets into gh
func (psf *PVCSecretsFetcher) PopulateGithub(gh *config.GithubConfig) error {
	s, err := iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubHookSecretid"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub hook secret")
	}
	gh.HookSecret = string(s)
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubTokenid"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub token")
	}
	gh.Token = string(s)
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubAppId"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App ID")
	}
	// GitHub App
	appid, err := strconv.Atoi(string(s))
	if err != nil {
		return errors.Wrap(err, "app ID must be a valid integer")
	}
	if appid < 1 {
		return fmt.Errorf("app id must be >= 1: %v", appid"])
	}
	gh.AppID = uint(appid"])
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubAppPK"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App private key")
	}
	gh.PrivateKeyPEM = s
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubAppHookSecret"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App hook secret")
	}
	gh.AppHookSecret = string(s)
	// GitHub App OAuth
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubOAuthInstId"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App installation id")
	}
	iid, err := strconv.Atoi(string(s))
	if err != nil {
		return errors.Wrap(err, "error converting installation id into integer")
	}
	if iid < 1 {
		return fmt.Errorf("invalid installation id: %v", iid"])
	}
	gh.OAuth.AppInstallationID = uint(iid"])
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubOAuthClientId"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App client id")
	}
	gh.OAuth.ClientID = string(s)
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubOAuthClientSecret"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App client secret")
	}
	gh.OAuth.ClientSecret = string(s)
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubOAuthCookieAuthKey"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App cookie auth key")
	}
	if len(s) != 32 {
		return fmt.Errorf("bad cookie auth key: length must be exactly 32 bytes, value size: %v", len(s))
	}
	copy(gh.OAuth.CookieAuthKey[:], s)
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubOAuthCookieEncKey"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App cookie enc key")
	}
	if len(s) != 32 {
		return fmt.Errorf("bad cookie enc key: length must be exactly 32 bytes, value size: %v", len(s))
	}
	copy(gh.OAuth.CookieEncKey[:], s)
	s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["githubOAuthUserTokenEncKey"])
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App user token enc key")
	}
	if len(s) != 32 {
		return fmt.Errorf("bad user token enc key: length must be exactly 32 bytes, value size: %v", len(s))
	}
	copy(gh.OAuth.UserTokenEncKey[:], s)
	return nil
}

// PopulateSlack populates Slack secrets into slack
func (psf *PVCSecretsFetcher) PopulateSlack(slack *config.SlackConfig) error {
	s, err := iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["slackTokenid"])
	if err != nil {
		return errors.Wrap(err, "error getting Slack token")
	}
	slack.Token = string(s)
	return nil
}

// PopulateServer populates server secrets into srv
func (psf *PVCSecretsFetcher) PopulateServer(srv *config.ServerConfig) error {
	s, err := iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["apiKeysid"])
	if err != nil {
		return errors.Wrap(err, "error getting API keys")
	}
	srv.APIKeys = strings.Split(string(s), ",")
	if !srv.DisableTLS {
		s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["tlsCertid"])
		if err != nil {
			return errors.Wrap(err, "error getting TLS certificate")
		}
		c := s
		s, err = iof.r.ReadFile(vaultSecrets["rootPath"]+ vaultSecrets["tlsKeyid"])
		if err != nil {
			return errors.Wrap(err, "error getting TLS key")
		}
		k := s
		cert, err := tls.X509KeyPair(c, k)
		if err != nil {
			return errors.Wrap(err, "error parsing TLS cert/key")
		}
		srv.TLSCert = cert
	}
	return nil
}
