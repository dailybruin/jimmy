package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	//"gopkg.in/mgo.v2/bson"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var GH_CLIENT_ID = os.Getenv("JIMMY_GH_CLIENT_ID")
var GH_CLIENT_SECRET = os.Getenv("JIMMY_GH_CLIENT_SECRET")
var SERVER_ENV = os.Getenv("JIMMY_SERVER_ENV")
var MONGO_URL = "0.0.0.0:27017"

func getRouter() (*mux.Router, error) {
	fs := http.FileServer(http.Dir("static"))
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/oauth_callback", oauthCallbackHandler)
	//r.Handle("/static/{(.+/?)*}", http.StripPrefix("/static/", fs))
	// Alternatively, use PathPrefix. However, PathPrefix will show directory of files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	return r, nil
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		/**
		   ** Check for valid path here
		  m := validPath.Find
		  if m == nil {
		      http.NotFound(w, r)
		      return
		  }
		  fn(w, r, m[2])
		  **/
		fn(w, r)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	type Template struct {
		Client_id string
	}

	t, _ := template.ParseFiles("templates/index.html")
	t.Execute(w, Template{
		Client_id: GH_CLIENT_ID,
	})
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func gitHubGet(access_token string) func(string, interface{}) error {
	return func(uri string, data interface{}) error {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com%s", uri), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", access_token))
		req.Header.Set("Authorization", fmt.Sprintf("token %s", access_token))

		client := &http.Client{}
		res, _ := client.Do(req)
		defer res.Body.Close()

		jsonDecodeErr := json.NewDecoder(res.Body).Decode(data)
		if jsonDecodeErr != nil {
			return jsonDecodeErr
		}
		return nil
	}
}

func oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	data := url.Values{}
	data.Set("client_id", GH_CLIENT_ID)
	data.Add("client_secret", GH_CLIENT_SECRET)
	data.Add("code", code)
	d := data.Encode()

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBufferString(d))
	if err != nil {
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(d)))

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	var OAuth struct {
		Access_token, Scope, Token_type string
	}
	jsonDecodeErr := json.NewDecoder(resp.Body).Decode(&OAuth)
	gitHubGet := gitHubGet(OAuth.Access_token)
	fmt.Println(jsonDecodeErr)

	fmt.Println(OAuth)

	_, MongoErr := mgo.Dial(MONGO_URL)
	if MongoErr != nil {
	}

	type Cookie struct {
		Value     string
		ExpiresOn string
	}
	type User struct {
		Login  string
		Cookie Cookie
	}
	//UsersCollection := session.DB("jimmy").C("users")
	//UsersCollection.Upsert(User{Username: ""})

	var authUser User
	gitHubGet("/user", &authUser)
	fmt.Println(authUser)
}
