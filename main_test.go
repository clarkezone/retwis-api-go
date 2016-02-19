package main

import (
	"bytes"
	"encoding/json"
	"github.com/clarkezone/jwtauth"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func Test(t *testing.T) {
}

func TestRetwisLogin(t *testing.T) {
	var provider = retwisUserProvider{}

	api := jwtauth.CreateApiSecurity(provider)

	ts := httptest.NewServer(http.HandlerFunc(api.Login))

	res, err := http.PostForm(ts.URL, url.Values{"username": {"foo"}, "password": {"bar"}})
	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fail()
	}

	result := GetBody(res)

	user := jwtauth.UserFromToken(result)
	if user != "1" {
		t.Fail()
	}
}

func TestGetTweets(t *testing.T) {
	var currentAuth jwtauth.JwtAuthProvider

	userid := "1"
	token, _ := currentAuth.GenerateToken(userid)

	ts := httptest.NewServer(http.HandlerFunc(jwtauth.RequireTokenAuthentication(GetTweets)))

	client := &http.Client{}

	req, _ := http.NewRequest("GET", ts.URL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	result, _ := client.Do(req)
	if result.StatusCode != http.StatusOK {
		t.Fail()
	}
}

func TestPostTweets(t *testing.T) {
	var currentAuth jwtauth.JwtAuthProvider

	userid := "1"
	token, _ := currentAuth.GenerateToken(userid)

	ts := httptest.NewServer(http.HandlerFunc(jwtauth.RequireTokenAuthentication(PostTweet)))

	client := &http.Client{}

	type Body struct {
		Post string
	}

	var b Body

	b.Post = "This is my test post"

	jsonBytes, err := json.Marshal(b)

	if err != nil {
		log.Fatal(err)
	}

	req, _ := http.NewRequest("GET", ts.URL, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	result, _ := client.Do(req)
	if result.StatusCode != http.StatusOK {
		t.Fail()
	}
}

func TestRegisterUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(RegisterUser))

	client := &http.Client{}

	type RegisterUser struct {
		Username string
		Password string
	}

	var b RegisterUser

	b.Username = "foo2"
	b.Password = "bar"

	jsonBytes, err := json.Marshal(b)

	if err != nil {
		log.Fatal(err)
	}

	req, _ := http.NewRequest("GET", ts.URL, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")

	result, _ := client.Do(req)
	if result.StatusCode != http.StatusOK {
		t.Fail()
	}
}

func GetBody(res *http.Response) (result string) {
	response, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	return string(response)
}
