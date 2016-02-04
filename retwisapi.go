package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
)

type RetwisApi struct {
	currentProvider userProvider
}

type userProvider interface {
	login(username string, password string) (result bool)
}

func (a *RetwisApi) RunServer() {
	http.HandleFunc("/login", a.Login)
	http.ListenAndServe(":8080", nil)
}

func CreateRetwisApi(p userProvider) (instance *RetwisApi) {
	r := new(RetwisApi)
	r.currentProvider = p
	return r
}

func (a *RetwisApi) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	fmt.Printf("Login username: %q password:%q", username, password)

	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "%q", html.EscapeString("username or password is empty"))
		return
	}
	if a.currentProvider.login(username, password) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%q", html.EscapeString("success"))
		fmt.Printf("Login username: %q password:%q is good", username, password)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "%q", html.EscapeString("fail"))
	}
}
