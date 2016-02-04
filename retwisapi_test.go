package main

import (
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type dummyUserProvider struct {
}

func (a dummyUserProvider) login(username string, password string) (result bool) {
	if username == "testuser" && password == "testpassword" {
		return true
	}
	return false
}

func TestLogin(t *testing.T) {
	var provider = dummyUserProvider{}

	api := CreateRetwisApi(provider)
	t.Log("Test Login")

	ts := httptest.NewServer(http.HandlerFunc(api.Login))

	res, err := http.PostForm(ts.URL, url.Values{"username": {"testuser"}, "password": {"testpassword"}})
	if err != nil {
		log.Fatal(err)
	}

	response, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	//trim off quotes
	responseString := string(response[1 : len(response)-1])
	if responseString == html.EscapeString("fail") {
		t.Fail()
	}
}
