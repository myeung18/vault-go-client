package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/myeung18/vault-go-client/pkg"
	"github.com/myeung18/vault-go-client/pkg/conjur"
	"log"
	"net/http"
)

func conjurVault(w http.ResponseWriter, r *http.Request) {

	client, err := conjur.NewConjurClient()
	if err != nil {
		fmt.Println("can't create conjur client", err)
	}
	fmt.Println("fetching Conjur secret. ")
	ids := []string{"os-climate/team1/awscredentials/aws-secretkey", ""}
	secret, errr := client.ReadSecret(ids[0])
	if errr != nil {
		fmt.Println("can't create Conjur client", errr)
	}

	fmt.Fprintf(w, "secret from Conjur: "+string(secret))
}

// not yet implemented
func conjurVaultSA(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "not yet implemented")
}

func main() {
	r := mux.NewRouter().StrictSlash(false) //exact '/' match is needed

	r.HandleFunc("/", indexHandler).Methods("GET")
	r.HandleFunc("/vault", pkg.AuthVaultAPI).Methods("GET")
	r.HandleFunc("/vault2", pkg.AuthVault).Methods("GET")
	r.HandleFunc("/conjur", conjurVault).Methods("GET")
	r.HandleFunc("/conjursa", conjurVaultSA).Methods("GET")

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	log.Println("Go vault client service is up..")
	server.ListenAndServe()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	pkg.SetHeader(w)
	fmt.Fprintf(w, "Hello World!")
}
