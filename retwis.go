package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/securecookie"
	"github.com/kylemcc/twitter-text-go/extract"
)

func isLogin(auth string) (*User, error) {

	if "" == auth {
		return nil, errors.New("No authentification token")
	}
	userId, err := redis.String(conn.Do("HGET", "auths", auth))
	if err != nil {
		return nil, err
	}
	savedAuth, err := redis.String(conn.Do("HGET", "user:"+userId, "auth"))
	if err != nil || savedAuth != auth {
		return nil, errors.New("Wrong authentification token")
	}
	return loadUserInfo(userId)
}

func loadUserInfo(userId string) (*User, error) {

	v, err := redis.Values(conn.Do("HGETALL", "user:"+userId))
	if err != nil {
		return nil, err
	}
	u := &User{Id: userId}
	err = redis.ScanStruct(v, u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func profileByUsername(username string) (*User, error) {

	if username == "" {
		return nil, errors.New("Invalid username")
	}
	userId, err := redis.String(conn.Do("HGET", "users", username))
	if err != nil {
		return nil, err
	}
	u := &User{Id: userId, Username: username}
	return u, nil
}

func profileByUserId(userId string) (*User, error) {

	if userId == "" {
		return nil, errors.New("Invalid user Id")
	}
	username, err := redis.String(conn.Do("HGET", "user:"+userId, "username"))
	if err != nil {
		return nil, err
	}
	u := &User{Id: userId, Username: username}
	return u, nil
}

func register(username, password string) (auth string, err error) {

	userId, err := redis.Int(conn.Do("INCR", "next_user_id"))
	if err != nil {
		return "", err
	}
	auth = string(securecookie.GenerateRandomKey(32)) // We reuse the securecookie random string generator
	auth, err = redis.String(registerScript.Do(
		conn,
		"users", // KEYS[1]
		fmt.Sprintf("user:%d", userId), // KEYS[2]
		"auths",            // KEYS[3]
		"users_by_time",    // KEYS[4]
		userId,             // ARGV[1]
		username,           // ARGV[2]
		password,           // ARGV[3]
		auth,               // ARGV[4]
		time.Now().Unix())) // ARGV[5]
	return auth, err
}

func login(username, password string) (auth string, err error, userid string) {

	userId, err := redis.Int(conn.Do("HGET", "users", username))
	if err != nil {
		return "", errors.New("Wrong username or password"), ""
	}
	realPassword, err := redis.String(conn.Do("HGET", fmt.Sprintf("user:%d", userId), "password"))
	if err != nil {
		return "", err, ""
	}
	if password != realPassword {
		return "", errors.New("Wrong username or password"), ""
	}
	auth, err = redis.String(conn.Do("HGET", fmt.Sprintf("user:%d", userId), "auth"))
	if err != nil {
		return "", err, ""
	}
	return auth, nil, strconv.Itoa(userId)
}

func logout(user *User) {

	if nil == user {
		return
	}

	newAuth := string(securecookie.GenerateRandomKey(32))
	oldAuth, _ := redis.String(conn.Do("HGET", "user:"+user.Id, "auth"))

	_, err := conn.Do("HSET", "user:"+user.Id, "auth", newAuth)
	if err != nil {
		log.Println(err)
	}
	_, err = conn.Do("HSET", "auths", newAuth, user.Id)
	if err != nil {
		log.Println(err)
	}
	_, err = conn.Do("HDEL", "auths", oldAuth)
	if err != nil {
		log.Println(err)
	}
}

func post(user *User, status string) error {

	postId, err := redis.Int(conn.Do("INCR", "next_post_id"))
	if err != nil {
		return err
	}
	status = strings.Replace(status, "\n", " ", -1)
	_, err = conn.Do("HMSET", fmt.Sprintf("post:%d", postId), "user_id", user.Id, "time", time.Now().Unix(), "body", status)
	if err != nil {
		return err
	}
	followers, err := redis.Strings(conn.Do("ZRANGE", "followers:"+user.Id, 0, -1))
	if err != nil {
		return err
	}
	recipients := mapset.NewSet()
	for _, fId := range followers {
		recipients.Add(fId)
	}
	entities := extract.ExtractMentionedScreenNames(status)
	for _, e := range entities {
		username, _ := e.ScreenName()
		profile, err := profileByUsername(username)
		if err == nil {
			recipients.Add(profile.Id)
		}
	}
	recipients.Add(user.Id)
	for fId := range recipients.Iter() {
		str, ok := fId.(string)
		if ok {
			conn.Do("LPUSH", "posts:"+str, postId)
		}
	}
	_, err = conn.Do("LPUSH", "timeline", postId)
	if err != nil {
		return err
	}
	_, err = conn.Do("LTRIM", "timeline", 0, 1000)
	if err != nil {
		return err
	}
	return nil
}

func strElapsed(t string) string {

	ts, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return ""
	}
	te := time.Now().Unix() - ts
	if te < 60 {
		return fmt.Sprintf("%d seconds", te)
	}
	if te < 3600 {
		m := int(te / 60)
		if m > 1 {
			return fmt.Sprintf("%d minutes", m)
		} else {
			return fmt.Sprintf("%d minute", m)
		}
	}
	if te < 3600*24 {
		h := int(te / 3600)
		if h > 1 {
			return fmt.Sprintf("%d hours", h)
		} else {
			return fmt.Sprintf("%d hour", h)
		}
	}
	d := int(te / (3600 * 24))
	if d > 1 {
		return fmt.Sprintf("%d days", d)
	} else {
		return fmt.Sprintf("%d day", d)
	}
}

func getPost(postId string) (*Post, error) {

	v, err := redis.Values(conn.Do("HGETALL", "post:"+postId))
	if err != nil {
		return nil, err
	}
	p := &Post{}
	err = redis.ScanStruct(v, p)
	if err != nil {
		return nil, err
	}
	username, err := redis.String(conn.Do("HGET", "user:"+p.UserId, "username"))
	if err != nil {
		return nil, err
	}
	p.Username = username
	p.Elapsed = strElapsed(p.Elapsed)
	return p, nil
}

/*
 getUserPosts returns posts of the timeline if key == "timeline"
 or of an user if key is something with this format posts:%d

 @return The posts, the number of remaining posts and an error if there is a problem
*/
func getUserPosts(key string, start, count int64) ([]*Post, int64, error) {

	values, err := redis.Strings(conn.Do("LRANGE", key, start, start+count-1))
	if err != nil {
		return nil, 0, err
	}
	posts := []*Post{}
	for _, pid := range values {
		p, err := getPost(pid)
		if err == nil {
			posts = append(posts, p)
		}
	}
	r, err := redis.Int64(conn.Do("LLEN", key))
	if err != nil {
		return posts, 0, nil
	} else {
		return posts, r - start - int64(len(values)), nil
	}
}

func getLastUsers() ([]*User, error) {

	v, err := redis.Strings(conn.Do("ZREVRANGE", "users_by_time", 0, 9))
	if err != nil {
		return nil, err
	}
	users := []*User{}
	for _, username := range v {
		users = append(users, &User{Username: username})
	}
	return users, nil
}
