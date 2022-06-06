package pkg

import (
	"fmt"
	"github.com/myeung18/vault-go-client/pkg/vault"
	"net/http"
	"os"
)

func AuthVault(w http.ResponseWriter, r *http.Request) {
	vaultAddr := os.Getenv("VAULT_ADDR")
	jwt := os.Getenv("JWT_PATH")

	fmt.Println(vaultAddr, jwt)
	client := vault.NewClient(vaultAddr, jwt)
	token := client.GetVaultToken()

	fmt.Println("token ", token)
	secret, err := client.RetrieveSecret(token)
	if err != nil {
		fmt.Println("failed to get secret ", err)
	}
	fmt.Println("secret ", secret)

	fmt.Fprintf(w, "token:"+secret["apikey"]+", cred:"+secret["name"])
}

func AuthVaultAPI(w http.ResponseWriter, r *http.Request) {
	SetHeader(w)
	secret, err := vault.AuthRetrieveSecret()
	if err != nil {
		fmt.Println("failed to auth&retrieve-secrect", err)
	}
	fmt.Fprintf(w, "Secret: "+"token:"+secret["apikey"]+", cred:"+secret["name"])
}

func SetHeader(w http.ResponseWriter) {
	// disable cache
	w.Header().Add("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "-1")
}
