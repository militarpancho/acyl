package secrets

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/dollarshaveclub/acyl/pkg/config"
	"github.com/dollarshaveclub/pvc"
	"github.com/pkg/errors"
)

// Secret IDs
// These will be interpolated as .ID in the secrets mapping
const (
	awsAccessKeyIDid           = "aws/access_key_id"
	awsSecretAccessKeyid       = "aws/secret_access_key"
	githubHookSecretid         = "github/hook_secret"
	githubTokenid              = "github/token"
	githubAppID                = "github/app/id"
	githubAppPK                = "github/app/private_key"
	githubAppHookSecret        = "github/app/hook_secret"
	githubOAuthInstID          = "github/app/oauth/installation_id"
	githubOAuthClientID        = "github/app/oauth/client/id"
	githubOAuthClientSecret    = "github/app/oauth/client/secret"
	githubOAuthCookieEncKey    = "github/app/oauth/cookie/encryption_key"
	githubOAuthCookieAuthKey   = "github/app/oauth/cookie/authentication_key"
	githubOAuthUserTokenEncKey = "github/app/oauth/user_token/encryption_key"
	apiKeysid                  = "api_keys"
	slackTokenid               = "slack/token"
	tlsCertid                  = "tls/cert"
	tlsKeyid                   = "tls/key"
	dbURIid                    = "db/uri"
)

type SecretFetcher interface {
	PopulateAllSecrets(aws *config.AWSCreds, gh *config.GithubConfig, slack *config.SlackConfig, srv *config.ServerConfig, pg *config.PGConfig) error
	PopulatePG(pg *config.PGConfig) error
	PopulateAWS(aws *config.AWSCreds) error
	PopulateGithub(gh *config.GithubConfig) error
	PopulateSlack(slack *config.SlackConfig) error
	PopulateServer(srv *config.ServerConfig) error
}

func PopulatePG(secretsBackend string, secretsConfig *config.SecretsConfig, vaultConfig *config.VaultConfig, pgConfig *config.PGConfig) error {
	sf, err := newSecretFetcher(secretsBackend, secretsConfig, vaultConfig)
	if err != nil {
		return errors.Wrapf(err, "secrets.PopulatePG error creating new secret fetcher")
	}
	err = sf.PopulatePG(pgConfig)
	if err != nil {
		return errors.Wrapf(err, "secrets.PopulatePG error setting pgConfig")
	}
	return nil
}

type PVCSecretsFetcher struct {
	sc *pvc.SecretsClient
}

func NewPVCSecretsFetcher(sc *pvc.SecretsClient) *PVCSecretsFetcher {
	return &PVCSecretsFetcher{
		sc: sc,
	}
}

// PopulateAllSecrets populates all secrets into the respective config structs
func (psf *PVCSecretsFetcher) PopulateAllSecrets(aws *config.AWSCreds, gh *config.GithubConfig, slack *config.SlackConfig, srv *config.ServerConfig, pg *config.PGConfig) error {
	if err := psf.PopulateAWS(aws); err != nil {
		return errors.Wrap(err, "error getting AWS secrets")
	}
	if err := psf.PopulateGithub(gh); err != nil {
		return errors.Wrap(err, "error getting GitHub secrets")
	}
	if err := psf.PopulateSlack(slack); err != nil {
		return errors.Wrap(err, "error getting Slack secrets")
	}
	if err := psf.PopulateServer(srv); err != nil {
		return errors.Wrap(err, "error getting server secrets")
	}
	if err := psf.PopulatePG(pg); err != nil {
		return errors.Wrap(err, "error getting db secrets")
	}
	return nil
}

// PopulatePG populates postgres secrets into pg
func (psf *PVCSecretsFetcher) PopulatePG(pg *config.PGConfig) error {
	s, err := psf.sc.Get(dbURIid)
	if err != nil {
		return errors.Wrap(err, "error getting DB URI")
	}
	pg.PostgresURI = string(s)
	return nil
}

// PopulateAWS populates AWS secrets into aws
func (psf *PVCSecretsFetcher) PopulateAWS(aws *config.AWSCreds) error {
	s, err := psf.sc.Get(awsAccessKeyIDid)
	if err != nil {
		return errors.Wrap(err, "error getting AWS access key ID")
	}
	aws.AccessKeyID = string(s)
	s, err = psf.sc.Get(awsSecretAccessKeyid)
	if err != nil {
		return errors.Wrap(err, "error getting AWS secret access key")
	}
	aws.SecretAccessKey = string(s)
	return nil
}

// PopulateGithub populates Github secrets into gh
func (psf *PVCSecretsFetcher) PopulateGithub(gh *config.GithubConfig) error {
	s, err := psf.sc.Get(githubHookSecretid)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub hook secret")
	}
	gh.HookSecret = string(s)
	s, err = psf.sc.Get(githubTokenid)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub token")
	}
	gh.Token = string(s)
	s, err = psf.sc.Get(githubAppID)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App ID")
	}
	// GitHub App
	appid, err := strconv.Atoi(string(s))
	if err != nil {
		return errors.Wrap(err, "app ID must be a valid integer")
	}
	if appid < 1 {
		return fmt.Errorf("app id must be >= 1: %v", appid)
	}
	gh.AppID = uint(appid)
	s, err = psf.sc.Get(githubAppPK)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App private key")
	}
	gh.PrivateKeyPEM = s
	s, err = psf.sc.Get(githubAppHookSecret)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App hook secret")
	}
	gh.AppHookSecret = string(s)
	// GitHub App OAuth
	s, err = psf.sc.Get(githubOAuthInstID)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App installation id")
	}
	iid, err := strconv.Atoi(string(s))
	if err != nil {
		return errors.Wrap(err, "error converting installation id into integer")
	}
	if iid < 1 {
		return fmt.Errorf("invalid installation id: %v", iid)
	}
	gh.OAuth.AppInstallationID = uint(iid)
	s, err = psf.sc.Get(githubOAuthClientID)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App client id")
	}
	gh.OAuth.ClientID = string(s)
	s, err = psf.sc.Get(githubOAuthClientSecret)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App client secret")
	}
	gh.OAuth.ClientSecret = string(s)
	s, err = psf.sc.Get(githubOAuthCookieAuthKey)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App cookie auth key")
	}
	if len(s) != 32 {
		return fmt.Errorf("bad cookie auth key: length must be exactly 32 bytes, value size: %v", len(s))
	}
	copy(gh.OAuth.CookieAuthKey[:], s)
	s, err = psf.sc.Get(githubOAuthCookieEncKey)
	if err != nil {
		return errors.Wrap(err, "error getting GitHub App cookie enc key")
	}
	if len(s) != 32 {
		return fmt.Errorf("bad cookie enc key: length must be exactly 32 bytes, value size: %v", len(s))
	}
	copy(gh.OAuth.CookieEncKey[:], s)
	s, err = psf.sc.Get(githubOAuthUserTokenEncKey)
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
	s, err := psf.sc.Get(slackTokenid)
	if err != nil {
		return errors.Wrap(err, "error getting Slack token")
	}
	slack.Token = string(s)
	return nil
}

// PopulateServer populates server secrets into srv
func (psf *PVCSecretsFetcher) PopulateServer(srv *config.ServerConfig) error {
	s, err := psf.sc.Get(apiKeysid)
	if err != nil {
		return errors.Wrap(err, "error getting API keys")
	}
	srv.APIKeys = strings.Split(string(s), ",")
	if !srv.DisableTLS {
		s, err = psf.sc.Get(tlsCertid)
		if err != nil {
			return errors.Wrap(err, "error getting TLS certificate")
		}
		c := s
		s, err = psf.sc.Get(tlsKeyid)
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

func newSecretFetcher(secretsBackend string, secretsConfig *config.SecretsConfig, vaultConfig *config.VaultConfig) (SecretFetcher, error) {
	if vaultConfig.UseAgent {
		sf := NewReadFileSecretsFetcher(vaultConfig)
		return sf, nil
	}
	ops := []pvc.SecretsClientOption{}
	switch secretsBackend {
	case "vault":
		secretsConfig.Backend = pvc.WithVaultBackend()
		switch {
		case vaultConfig.TokenAuth:
			log.Printf("secrets: using vault token auth")
			ops = []pvc.SecretsClientOption{
				pvc.WithVaultAuthentication(pvc.Token),
				pvc.WithVaultToken(vaultConfig.Token),
			}
		case vaultConfig.K8sAuth:
			log.Printf("secrets: using vault k8s auth")
			jwt, err := ioutil.ReadFile(vaultConfig.K8sJWTPath)
			if err != nil {
				errors.Wrapf(err, "error reading k8s jwt path: %v", err)
			}
			log.Printf("secrets: role: %v; auth path: %v", vaultConfig.K8sRole, vaultConfig.K8sAuthPath)
			ops = []pvc.SecretsClientOption{
				pvc.WithVaultAuthentication(pvc.K8s),
				pvc.WithVaultK8sAuth(string(jwt), vaultConfig.K8sRole),
				pvc.WithVaultK8sAuthPath(vaultConfig.K8sAuthPath),
			}
		case vaultConfig.AppID != "" && vaultConfig.UserIDPath != "":
			log.Printf("secrets: using vault AppID auth")
			ops = []pvc.SecretsClientOption{
				pvc.WithVaultAuthentication(pvc.AppID),
				pvc.WithVaultAppID(vaultConfig.AppID),
				pvc.WithVaultUserIDPath(vaultConfig.UserIDPath),
			}
		default:
			errors.New("no Vault authentication methods were supplied")
		}
		ops = append(ops, pvc.WithVaultHost(vaultConfig.Addr))
	case "env":
		secretsConfig.Backend = pvc.WithEnvVarBackend()
	default:
		errors.New(fmt.Sprintf("invalid secrets backend: %v", secretsBackend))
	}
	if secretsConfig.Mapping == "" {
		errors.New("secrets mapping is required")
	}
	ops = append(ops, pvc.WithMapping(secretsConfig.Mapping), secretsConfig.Backend)
	sc, err := pvc.NewSecretsClient(ops...)
	if err != nil {
		return nil, errors.Wrapf(err, "secrets.getSecretsClient error creating new PVC Secrets Client")
	}
	sf := NewPVCSecretsFetcher(sc)
	return sf, nil
}

func getSecrets() error {
	if vaultConfig.UseAgent {
		sf := NewReadFileSecretsFetcher(&vaultConfig)
		err := sf.PopulateAllSecrets(&awsCreds, &githubConfig, &slackConfig, &serverConfig, &pgConfig)
		if err != nil {
			return errors.Wrapf("secrets.getSecrets error populating secrets using Vault Agent Injector : %v", err)
		}
		return nil
	}
	sc, err := getSecretClient()
	if err != nil {
		errors.Wrapf(err, "error getting secrets client: %v", err)
	}
	sf := secrets.NewPVCSecretsFetcher(sc)
	err = sf.PopulateAllSecrets(&awsCreds, &githubConfig, &slackConfig, &serverConfig, &pgConfig)
	if err != nil {
		errors.Wrapf(err, "error getting secrets: %v", err)
	}
}
