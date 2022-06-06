package vault

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/levigross/grequests"
	"github.com/myeung18/vault-go-client/pkg/k8s"

	"io"
	"io/ioutil"
	"os"
)

type VaultClient struct {
	vaultAddr string
	jwt       string
}

func NewClient(addr string, jwt string) *VaultClient {
	return &VaultClient{vaultAddr: addr, jwt: jwt}
}

func ParseSecret(r io.Reader) (*api.Secret, error) {
	// First read the data into a buffer. Not super efficient but we want to
	// know if we actually have a body or not.
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return nil, nil
	}

	// First decode the JSON into a map[string]interface{}
	var secret api.Secret
	if err := jsonutil.DecodeJSONFromReader(&buf, &secret); err != nil {
		return nil, err
	}

	return &secret, nil
}

func (vc *VaultClient) RetrieveSecret(token string) (map[string]string, error) {

	url := fmt.Sprintf("%v/"+SecretPath, vc.vaultAddr)
	opt := buildReqOptions()
	opt.Headers["X-Vault-Token"] = token

	resp, err := grequests.Get(url, opt)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	secret := new(api.Secret)
	err = resp.JSON(secret)
	if err != nil {
		fmt.Println("failed to get credential", err)
		return nil, err
	}

	//secret, err := ParseSecret(resp.RawResponse.Body)
	fmt.Println("secret", secret)
	var val string
	for k, v := range secret.Data {
		val += k + ", "
		fmt.Println(k, v)
	}
	fmt.Println("cred: ", val)
	fmt.Println("secret.data:", secret.Data["data"])

	return retrieveSecret(secret), nil
}

func (vc *VaultClient) GetVaultToken() string {
	sa, role := os.Getenv("VAULT_SERVICE_ACCT"), os.Getenv("VAULT_ROLE")
	ns := os.Getenv("USER_NAMESPACE")

	jwtString, err := k8s.ReadServiceAccountToken(sa, ns)
	//jwtString, err := LookupJwt(&vc.jwt)
	if err != nil {
		fmt.Println("error reading jwt", err)
		return ""
	}
	fmt.Println("jwt read from sa: ", jwtString)

	varReq := VaultReq{
		Role: role,
		Jwt:  jwtString,
	}
	opt := buildReqOptions()
	opt.JSON = varReq

	url := fmt.Sprintf("%v/"+AuthPath, vc.vaultAddr)
	fmt.Println("sending vault token request.", url)
	resp, err := grequests.Put(url, opt)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	fmt.Println("login res:", resp.String())
	vault := new(VaultRep)
	err = resp.JSON(vault)
	if err != nil {
		fmt.Println("failed to get vault rep body", err)
		return ""
	}

	return vault.Auth.ClientToken
	//sec := vc.RetrieveSecret(token)
	//return token + ":" + sec
}

func buildReqOptions() *grequests.RequestOptions {
	return &grequests.RequestOptions{
		Headers: map[string]string{
			"Content-Type": "application/json",
			//"X-Vault-Token": token,
		},
		InsecureSkipVerify: true,
	}
}

func getJwtFromSecret() (string, error) {

	return "", nil
}

func LookupJwt(tokenPath *string) (string, error) {
	buf, err := ioutil.ReadFile(*tokenPath)
	if err != nil {
		return "", err
	}

	s := string(buf)
	return s, nil
}

type VaultReq struct {
	Role string `json:"role,omitempty"`
	Jwt  string `json:"jwt,omitempty"`
}

type Auth struct {
	ClientToken string `json:"client_token,omitempty"`
}

type VaultRep struct {
	Auth      Auth `json:"auth,omitempty"`
	Renewable bool `json:"renewable,omitempty"`
}

var AuthPath = "v1/auth/kubernetes/login"
var SecretPath = "v1/secret/data/webapp/config"

func AuthRetrieveSecret() (map[string]string, error) {

	sa, role := os.Getenv("VAULT_SERVICE_ACCT"), os.Getenv("VAULT_ROLE")
	ns := os.Getenv("USER_NAMESPACE")

	vaultAddr := os.Getenv("VAULT_ADDR")
	jwt := os.Getenv("JWT_PATH")

	vConfig := api.DefaultConfig()
	vConfig.Address = vaultAddr

	os.Setenv(api.EnvVaultSkipVerify, "true")

	fmt.Println("config-skip", vaultAddr, jwt, vConfig)

	client, err := api.NewClient(vConfig)
	if err != nil {
		fmt.Println("failed to create client", err)
		return nil, err
	}
	//jwtString, err := LookupJwt(&jwt)
	jwtString, err := k8s.ReadServiceAccountToken(sa, ns)
	if err != nil {
		fmt.Println("error reading jwt", err)
		return nil, err
	}
	fmt.Println("jwt string from sa", jwtString)
	url := AuthPath
	varReq := VaultReq{
		Role: sa,
		Jwt:  jwtString,
	}
	fmt.Println("sending vault token request.", url, varReq)

	k8sAuth, err := auth.NewKubernetesAuth(
		role,
		auth.WithServiceAccountToken(jwtString))

	if err != nil {
		fmt.Println("failed to new k8s auth", err)
	}

	authInfo, err := client.Auth().Login(context.TODO(), k8sAuth)
	if err != nil {
		fmt.Println("failed to new k8s auth", err)
	}

	fmt.Println("authninfo: ", authInfo)
	SecretPath = "secret/data/webapp/config"
	cred, errr := client.Logical().Read(SecretPath)
	if errr != nil {
		fmt.Println("failed to retrieve credential", err)
		return nil, errr
	}

	return retrieveSecret(cred), nil
}

func retrieveSecret(secret *api.Secret) map[string]string {
	seMap := secret.Data["data"].(map[string]interface{})
	res := make(map[string]string)
	pwd, ok := seMap["password"].(string)
	if !ok {
		fmt.Errorf("password not found")
	}
	res["apikey"] = pwd
	userName, ok := seMap["username"].(string)
	if !ok {
		fmt.Errorf("username not found")
	}
	res["name"] = userName

	return res
}
