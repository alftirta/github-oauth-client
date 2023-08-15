package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	// load .env in the current path
	if err := godotenv.Load(); err != nil {
		log.Fatalf(".env not found, err: %q", err)
	}
}

func main() {
	// root
	http.HandleFunc("/", rootHandler)

	// github login
	http.HandleFunc("/login/github", githubLoginHandler)

	// github callback
	http.HandleFunc("/login/github/callback", githubCallbackHandler)

	// serve http
	port := getEnvValue("PORT")
	log.Printf("server running on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func getEnvValue(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("missing %s in .env", key)
	}
	return value
}

func getGithubAccessToken(code string) string {
	clientID := getEnvValue("GITHUB_CLIENT_ID")
	clientSecret := getEnvValue("GITHUB_CLIENT_SECRET")

	// POST request
	jsonRequestBody := []byte(
		fmt.Sprintf(
			`{"client_id":%q,"client_secret":%q,"code":%q}`,
			clientID,
			clientSecret,
			code,
		),
	)
	req, err := http.NewRequest(
		http.MethodPost,
		"https://github.com/login/oauth/access_token",
		bytes.NewBuffer(jsonRequestBody),
	)
	if err != nil {
		log.Fatalf("create new request failed, err: %q", err)
	}

	/*
		Alternatively, you can request for access token by sending a GET method request.
		Uncomment the below codes to try out.
	*/
	// req, err := http.NewRequest(
	// 	http.MethodGet,
	// 	fmt.Sprintf(
	// 		"https://github.com/login/oauth/access_token?grant_type=%s&client_id=%s&client_secret=%s&code=%s",
	// 		"authorization_code",
	// 		clientID,
	// 		clientSecret,
	// 		code,
	// 	),
	// 	nil,
	// )
	// if err != nil {
	// 	log.Fatalf("create new request failed, err: %q", err)
	// }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("send http request failed, err: %q", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("io read all failed, err: %q", err)
	}

	type githubAccessTokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	var gatr githubAccessTokenResponse
	if err := json.Unmarshal(b, &gatr); err != nil {
		log.Fatalf("json unmarshal failed, err: %q", err)
	}

	return gatr.AccessToken
}

func getGithubUserData(accessToken string) []byte {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		log.Fatalf("create new request failed, err: %q", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("send http request failed, err: %q", err)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("io read all failed, err: %q", err)
	}

	return b
}

func displayUserData(w http.ResponseWriter, r *http.Request, userData []byte) {
	// validate user data
	if len(userData) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "unauthorized")
		return
	}

	// display user data
	w.Header().Set("Content-Type", "application/json")
	var jsonResponse bytes.Buffer
	if err := json.Indent(&jsonResponse, userData, "", "\t"); err != nil {
		log.Fatalf("json indent failed, err: %q", err)
	}
	fmt.Fprint(w, jsonResponse.String())
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<a href="/login/github">Login with GitHub Account</a>`)
}

func githubLoginHandler(w http.ResponseWriter, r *http.Request) {
	githubClientID := getEnvValue("GITHUB_CLIENT_ID")
	redirectURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s",
		githubClientID,
		fmt.Sprintf(
			"%s://%s:%s/login/github/callback",
			getEnvValue("PROTOCOL"),
			getEnvValue("HOST"),
			getEnvValue("PORT"),
		),
	)
	http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
}

func githubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	accessToken := getGithubAccessToken(code)
	userData := getGithubUserData(accessToken)
	displayUserData(w, r, userData)
}
