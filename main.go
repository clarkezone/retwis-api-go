package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/clarkezone/jwtauth"
	"io/ioutil"
	"net/http"
)

type retwisUserProvider struct {
}

type errorObj struct {
	Message string
}

func hasFailed(wr http.ResponseWriter, err error) bool {
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)

		o := errorObj{err.Error()}

		buffer, _ := json.Marshal(o)

		wr.Write(buffer) //ignore errors

		return true
	}
	return false
}

func (a retwisUserProvider) Login(username string, password string) (result bool, userid string) {
	_, err, id := login(username, password)
	if err != nil {
		fmt.Printf("Login:Error:%v", err.Error())
		return false, ""
	}

	fmt.Printf("Login:%v\n", username)
	return true, id
}

func RegisterUser(wr http.ResponseWriter, r *http.Request) {
	type RegisterUser struct {
		Username string
		Password string
	}

	var b RegisterUser

	bodyBytes, err := ioutil.ReadAll(r.Body)

	if hasFailed(wr, err) {
		return
	}

	err = json.Unmarshal(bodyBytes, &b)

	if hasFailed(wr, err) {
		return
	}

	if b.Username == "" || b.Password == "" {
		hasFailed(wr, errors.New("Username or Password is empty or invalid."))
		return
	}

	_, err = register(b.Username, b.Password)

	if hasFailed(wr, err) {
		fmt.Printf("Error:" + err.Error())
		return
	}
}

func GetTweets(wr http.ResponseWriter, r *http.Request) {
	fmt.Printf("Get Tweets\n")
	userid := r.Header.Get("userid")
	user, _ := loadUserInfo(userid)
	if user != nil {
		wr.Header().Set("Content-Type", "application/json")
		posts, _, _ := getUserPosts("posts:"+user.Id, 0, 10)
		theJson, _ := json.Marshal(posts)
		fmt.Fprintf(wr, string(theJson))
	}
}

func PostTweet(wr http.ResponseWriter, r *http.Request) {
	fmt.Printf("Post Tweet\n")
	userid := r.Header.Get("userid")
	user, err := loadUserInfo(userid)
	if hasFailed(wr, err) {
		return
	}

	type Body struct {
		Post string
	}

	var b Body

	bodyBytes, err := ioutil.ReadAll(r.Body)

	if hasFailed(wr, err) {
		return
	}
	err = json.Unmarshal(bodyBytes, &b)

	if hasFailed(wr, err) {
		fmt.Printf("Failed to unmarshall" + err.Error())
		return
	}
	err = post(user, b.Post)

	if hasFailed(wr, err) {
		return
	}
}

func main() {
	var provider = retwisUserProvider{}
	api := jwtauth.CreateApiSecurity(provider)
	api.RegisterAuthHandlers()
	http.HandleFunc("/tweets", jwtauth.RequireTokenAuthentication(GetTweets))
	http.HandleFunc("/post", jwtauth.RequireTokenAuthentication(PostTweet))
	http.HandleFunc("/register", RegisterUser)
	http.ListenAndServe(":8080", nil)
}
