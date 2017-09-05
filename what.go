package what

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	"os"
)

const (
	defaultLayout = "templates/layout.html"
	templateDir   = "templates/"

	defaultConfigFile = "config.json"

	githubAuthorizeUrl = "https://github.com/login/oauth/authorize"
	githubTokenUrl     = "https://github.com/login/oauth/access_token"
	redirectUrl        = ""
)

type Config struct {
	ClientSecret string `json:"clientSecret"`
	ClientID     string `json:"clientID"`
	Secret       string `json:"secret"`
}

var (
	cfg      *Config
	oauthCfg *oauth2.Config
	store    *sessions.CookieStore

	// scopes
	scopes = []string{"repo"}

	tmpls = map[string]*template.Template{}
)

func loadConfig(file string) (*Config, error) {
	var config Config

	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func Init() {
	tmpls["home.html"] = template.Must(
		template.ParseFiles(templateDir+"home.html", defaultLayout),
	)
	tmpls["prs.html"] = template.Must(
		template.ParseFiles(templateDir+"prs.html", defaultLayout),
	)

	var err error
	cfg, err = loadConfig(defaultConfigFile)
	if err != nil {
		panic(err)
	}

	store = sessions.NewCookieStore([]byte(cfg.Secret))

	oauthCfg = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  githubAuthorizeUrl,
			TokenURL: githubTokenUrl,
		},
		RedirectURL: redirectUrl,
		Scopes:      scopes,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/start", StartHandler)
	r.HandleFunc("/auth", CallbackHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", r)

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	log.Fatalln(http.ListenAndServe(":"+port, nil))
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "sess")
	if err != nil {
		fmt.Fprintln(w, "aborted")
		return
	}
	tokenObj, ok := session.Values["accessToken"]
	if tokenObj == nil || !ok {
		tmpls["home.html"].ExecuteTemplate(w, "base", map[string]interface{}{})
		return
	}

	accessToken := tokenObj.(string)
	tkn := oauth2.Token{AccessToken: accessToken}

	data, err := githubData(&tkn)
	if err != nil {
		fmt.Fprintln(w, "aborted")
		return
	}

	tmpls["prs.html"].ExecuteTemplate(w, "base", data)

}

func StartHandler(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	rand.Read(b)

	state := base64.URLEncoding.EncodeToString(b)

	session, _ := store.Get(r, "sess")
	session.Values["state"] = state
	session.Save(r, w)

	url := oauthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, 302)
}

type GithubData struct {
	UserPRs          []PullRequest
	ParticipatingPRs []PullRequest
}

var EmptyGithubData = GithubData{}

func githubData(tkn *oauth2.Token) (GithubData, error) {
	client := github.NewClient(oauthCfg.Client(oauth2.NoContext, tkn))
	var oh = map[string]string{
		"query": query,
	}
	newReq, err := client.NewRequest("POST", "https://api.github.com/graphql", oh)
	if err != nil {
		return EmptyGithubData, err
	}

	var res GithubResponse
	_, err = client.Do(oauth2.NoContext, newReq, &res)

	if err != nil {
		return EmptyGithubData, err
	}
	r := GithubData{
		UserPRs:          res.UserPRs(),
		ParticipatingPRs: res.ParticipatingPRs(),
	}
	return r, nil
}

func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "sess")
	if err != nil {
		fmt.Fprintln(w, "aborted")
		return
	}

	if r.URL.Query().Get("state") != session.Values["state"] {
		fmt.Fprintln(w, "no state match; possible csrf OR cookies not enabled")
		return
	}

	tkn, err := oauthCfg.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
	if err != nil {
		fmt.Fprintln(w, "there was an issue getting your token")
		return
	}

	if !tkn.Valid() {
		fmt.Fprintln(w, "retreived invalid token")
		return
	}
	session.Values["accessToken"] = tkn.AccessToken
	session.Save(r, w)

	http.Redirect(w, r, "/", 302)
}
