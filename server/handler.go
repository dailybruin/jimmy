package main

import (
	"bytes"
	//"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
	r.HandleFunc("/dashboard", dashHandler)
	r.HandleFunc("/repo/{repo}", trackRepoHandler).Methods("POST")
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

		client := &http.Client{}
		res, reqErr := client.Do(req)
		if reqErr != nil {
			return reqErr
		}
		defer res.Body.Close()

		jsonDecodeErr := json.NewDecoder(res.Body).Decode(data)
		if jsonDecodeErr != nil {
			return jsonDecodeErr
		}
		return nil
	}
}

func gitHubPost(access_token string) func(string, interface{}) (*http.Response, error) {
	return func(uri string, params interface{}) (*http.Response, error) {
		json, jsonMarshalErr := json.Marshal(params)
		if jsonMarshalErr != nil {
			return nil, jsonMarshalErr
		}
		fmt.Println(string(json))
		req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com%s", uri), bytes.NewBuffer(json))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", access_token))
		req.Header.Add("Content-Type", "application/json")
		client := http.Client{}
		res, reqErr := client.Do(req)
		if reqErr != nil {
			return nil, reqErr
		}
		return res, nil
	}
}

type User struct {
	Login        string
	Email        string
	Cookie       string
	Access_token string
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

	session, mongoErr := mgo.Dial(MONGO_URL)
	if mongoErr != nil {
	}

	var authUser User
	gitHubGet("/user", &authUser)
	//hash := md5.Sum([]byte(fmt.Sprintf("%v%v", authUser.Login, authUser.Email)))
	authUser.Access_token = OAuth.Access_token
	//authUser.Cookie = string(hash[:16])
	authUser.Cookie = fmt.Sprintf("%v%v", authUser.Login, authUser.Email)
	cookie := http.Cookie{
		Name:  "DBJimmyAuth",
		Value: authUser.Cookie,
	}
	http.SetCookie(w, &cookie)

	usersCollection := session.DB("jimmy").C("users")
	usersCollection.Upsert(bson.M{"login": authUser.Login}, authUser)
	http.Redirect(w, r, "/dashboard", 303)
}

func getAuth(w http.ResponseWriter, r *http.Request, c chan *User) {
	cookie, cookieErr := r.Cookie("DBJimmyAuth")
	if cookieErr != nil {
		fmt.Println("Cookie Error")
		c <- nil
	}
	session, mongoErr := mgo.Dial(MONGO_URL)
	if mongoErr != nil {
		fmt.Println("Mongo Error")
		c <- nil
	}
	var user User
	usersCollection := session.DB("jimmy").C("users")
	queryErr := usersCollection.Find(bson.M{"cookie": cookie.Value}).One(&user)
	if queryErr != nil {
		fmt.Println("Query Error")
		c <- nil
	}
	c <- &user
}

func dashHandler(w http.ResponseWriter, r *http.Request) {
	c := make(chan *User)
	go getAuth(w, r, c)
	user := <-c
	gitHubGet := gitHubGet(user.Access_token)
	type Org struct {
		Login       string `json:"login"`
		Id          int    `json:"id"`
		Url         string `json:"url"`
		Avatar_url  string `json:"avatar_url"`
		Description string `json:"description"`
	}
	type Orgs struct {
		Org []Org `json:"array"`
	}
	var orgs []Org
	err := gitHubGet("/user/orgs", &orgs)
	if err != nil {
		fmt.Println("get orgs error")
		fmt.Println(err)
	}
	fmt.Println(orgs)
	var repos []interface{}
	for _, v := range orgs {
		if v.Login == "daily-bruin" {
			repoGetErr := gitHubGet("/orgs/daily-bruin/repos", &repos)
			if repoGetErr != nil {
				fmt.Println(repoGetErr)
			}
			break
		}
	}
	fmt.Println(repos)
	var data struct {
		Repos []interface{}
	}
	data.Repos = repos
	t, _ := template.ParseFiles("templates/dashboard.html")
	t.Execute(w, data)
}

func trackRepoHandler(w http.ResponseWriter, r *http.Request) {
	c := make(chan *User)
	go getAuth(w, r, c)
	user := <-c
	gitHubPost := gitHubPost(user.Access_token)

	vars := mux.Vars(r)
	repo := vars["repo"]
	var data struct {
		Name   string      `json:"name"`
		Config interface{} `json:"config"`
		Events []string    `json:"events"`
		Active bool        `json:"active"`
	}
	data.Name = "web"
	type Config struct {
		Url          string `json:"url"`
		Content_type string `json:"content_type"`
		Secret       string `json:"secret"`
		Insecure_ssl string `json:"insecure_ssl"`
	}
	var config Config
	config.Url = "http://localhost"
	config.Content_type = "json"
	config.Secret = "SOME_SECRET_YOU_SHOULD_PROBABLY_CHANGE"
	config.Insecure_ssl = "0"
	data.Config = config
	data.Events = []string{"pull_request"}
	data.Active = true
	gitHubPostRes, gitHubPostErr := gitHubPost(fmt.Sprintf("/repos/daily-bruin/%s/hooks", repo), data)
	if gitHubPostErr != nil {
		fmt.Println(gitHubPostErr)
	}
	fmt.Printf("response: %v", gitHubPostRes)
	var webHookRes struct {
		Id     int    `json:"id"`
		Name   string `json:"name"`
		Active bool   `json:"active"`
		Config Config `json:"config"`
	}
	jsonDecodeErr := json.NewDecoder(gitHubPostRes.Body).Decode(&webHookRes)

	// should consider moving this to gitHubPost like gitHubGet
	defer gitHubPostRes.Body.Close()

	if jsonDecodeErr != nil {
		fmt.Println(jsonDecodeErr)
	}
}
