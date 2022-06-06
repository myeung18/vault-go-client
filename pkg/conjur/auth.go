package conjur

import (
	"errors"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"

	//"github.com/cyberark/conjur-api-go/conjurapi/authn"
	"github.com/cyberark/conjur-authn-k8s-client/pkg/authenticator"
	authnConfig "github.com/cyberark/conjur-authn-k8s-client/pkg/authenticator/config"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const authenticatorTokenFile = "/run/conjur/access-token"

type ConjurVaultClient struct {
	AuthenticationMutex *sync.Mutex
	Authenticator       authenticator.Authenticator
	AuthenticatorConfig authnConfig.Configuration
	Config              conjurapi.Config
	Conjur              *conjurapi.Client
	Version             string
	Name                string

	// Credentials for API-key based auth
	APIKey   string
	Username string

	// Authn URL for K8s-authenticator based auth
	AuthnURL string
}

func NewConjurClient() (*ConjurVaultClient, error) {

	config, err := conjurapi.LoadConfig()

	fmt.Println("config >> : ", config)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Conjur provider could not load configuration: %s", err)
	}

	var apiKey, authnURL, username, version string
	var conjurAuthenticator authenticator.Authenticator
	var conjurAuthenticatorConf authnConfig.Configuration
	var conjur *conjurapi.Client

	authenticationMutex := &sync.Mutex{}

	os.Setenv("DEBUG", "true")
	username = os.Getenv("CONJUR_AUTHN_LOGIN")
	apiKey = os.Getenv("CONJUR_AUTHN_API_KEY")
	authnURL = os.Getenv("CONJUR_AUTHN_URL")
	version = os.Getenv("CONJUR_VERSION")

	fmt.Println("config - env >> : ", username,",", apiKey,",", authnURL,",", version)

	if len(version) == 0 {
		version = "5"
	}
	client := &ConjurVaultClient{
		Name:                "conjur",
		Config:              config,
		Username:            username,
		AuthenticatorConfig: conjurAuthenticatorConf,
		APIKey:              apiKey,
		AuthnURL:            authnURL,
		AuthenticationMutex: authenticationMutex,
		Version:             version,
	}

	conjurAuthenticatorConf, err = authnConfig.NewConfigFromEnv()
	fmt.Println(fmt.Sprintf("authnConfig: %s", conjurAuthenticatorConf))
	if err != nil {
		return nil, err
	}
	conjurAuthenticator, err = authenticator.NewAuthenticator(conjurAuthenticatorConf)

	if err != nil {
		return nil, fmt.Errorf("ERROR: Conjur provider could not retrieve access token using the authenticator client: %s", err)
	}
	client.Authenticator = conjurAuthenticator
	client.AuthenticatorConfig = conjurAuthenticatorConf

	fmt.Println("Fetching AccessToken........")
	refreshErr := client.fetchAccessToken()
	if refreshErr != nil {
		return nil, refreshErr
	}

	fmt.Println("Fetching AccessToken Loop........")
	go func() {
		// Sleep until token needs refresh
		time.Sleep(client.AuthenticatorConfig.GetTokenTimeout())

		fmt.Println("Fetching AccessToken Loop Timeout........")
		// Authenticate in a loop
		err := client.fetchAccessTokenLoop()

		// On repeated errors in getting the token, we need to exit the
		// broker since the provider cannot be used.
		if err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Println("getting conjur authn token.")
	// Once the token file has been loaded, create a new instance of the Conjur client
	if conjur, err = conjurapi.NewClientFromTokenFile(client.Config, authenticatorTokenFile); err != nil {
		return nil, fmt.Errorf("ERROR: Could not create new Conjur provider: %s", err)
	}
	client.Conjur = conjur
	return client, nil
}

func (v ConjurVaultClient) fetchAccessToken() error {
	// Configure exponential backoff
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 2 * time.Second
	expBackoff.RandomizationFactor = 0.5
	expBackoff.Multiplier = 2
	expBackoff.MaxInterval = 15 * time.Second
	expBackoff.MaxElapsedTime = 2 * time.Minute

	// Authenticate with retries on failure with exponential backoff
	err := backoff.Retry(func() error {
		// Lock the authenticatorMutex
		v.AuthenticationMutex.Lock()
		defer v.AuthenticationMutex.Unlock()

		log.Printf("Info: Conjur provider is authenticating ...")
		if err := v.Authenticator.Authenticate(); err != nil {
			log.Printf("Info: Conjur provider received an error on authenticate: %s", err.Error())
			return err
		}

		log.Printf("Info: Conjur provider is authenticating ...done ")
		return nil
	}, expBackoff)

	if err != nil {
		return fmt.Errorf("error: Conjur provider unable to authenticate; backoff exhausted: %s", err.Error())
	}

	return nil
}

func (v ConjurVaultClient) fetchAccessTokenLoop() error {
	if v.Authenticator == nil {
		return errors.New("error: Conjur Kubernetes authenticator must be instantiated before access token may be refreshed")
	}

	// Fetch the access token in a loop
	for {
		err := v.fetchAccessToken()
		if err != nil {
			return err
		}

		// sleep until token needs refresh
		time.Sleep(v.AuthenticatorConfig.GetTokenTimeout())
	}
}

func (v ConjurVaultClient) ReadSecret(id string) ([]byte, error) {
	var err error

	if id == "accessToken" {
		if v.Username != "" && v.APIKey != "" {
			// TODO: Use a cached access token from the client, once it's exposed
			return v.Conjur.Authenticate(authn.LoginPair{
				v.Username,
				v.APIKey,
			})
		}
		return nil, errors.New("error: Conjur provider can't provide an accessToken unless username and apiKey credentials are provided")
	}

	// If using the Conjur Kubernetes authenticator, ensure that the
	// Conjur API is using the current access token
	if v.AuthnURL != "" {
		if v.Conjur, err = conjurapi.NewClientFromTokenFile(v.Config, authenticatorTokenFile); err != nil {
			log.Fatalf("ERROR: Could not create new Conjur provider: %s", err)
		}
	}
	fmt.Println("Id to fetch", id)

	tokens := strings.SplitN(id, ":", 3)
	switch len(tokens) {
	case 1:
		tokens = []string{v.Config.Account, "variable", tokens[0]}
	case 2:
		tokens = []string{v.Config.Account, tokens[0], tokens[1]}
	}

	return v.Conjur.RetrieveSecret(strings.Join(tokens, ":"))
}
