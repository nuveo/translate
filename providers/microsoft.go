package microsoft

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	datamarket    = "https://datamarket.accesscontrol.windows.net/v2/OAuth2-13"
	scope         = "http://api.microsofttranslator.com"
	translate_url = "http://api.microsofttranslator.com/v2/Http.svc/Translate"
	grant_type    = "client_credentials"
)

type TokenRequest struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type,omitempty"`
	ExpiresIn   string    `json:"expires_in"`
	Scope       string    `json:"scope"`
	Timeout     time.Time `json:"timeout"`
}

// Make a POST request to `datamark` url
func (t *TokenRequest) GetAccessToken() TokenResponse {

	client := &http.Client{}

	postValues := url.Values{}
	postValues.Add("client_id", t.ClientId)
	postValues.Add("client_secret", t.ClientSecret)
	postValues.Add("scope", scope)
	postValues.Add("grant_type", grant_type)

	req, err := http.NewRequest("POST", datamarket, strings.NewReader(postValues.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		log.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var tr TokenResponse
	err = json.Unmarshal(body, &tr)
	if err != nil {
		log.Println(err)
	}

	now := time.Now()
	expires_in, err := strconv.ParseInt(tr.ExpiresIn, 10, 0)
	if err != nil {
		log.Println(err)
	}

	exp_time := now.Add(time.Duration(expires_in) * time.Second)
	// 10 min
	tr.Timeout = exp_time
	return tr
}

func (t *TokenResponse) Translate(text, from, to string) (string, error) {

	if t.CheckTimeout() == true {
		return "", errors.New("Access token is invalid, please get new token")
	}

	client := &http.Client{}

	textEncode := url.Values{}
	textEncode.Add("text", text)
	text = textEncode.Encode()

	url := fmt.Sprintf("%s?%s&from=%s&to=%s", translate_url, text, from, to)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	auth_token := fmt.Sprintf("Bearer %s", t.AccessToken)
	req.Header.Add("Authorization", auth_token)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	type Text struct {
		T string `xml:",chardata"`
	}

	var obj Text
	err = xml.Unmarshal(body, &obj)

	return obj.T, nil
}

// Verify if access token is valid
func (t *TokenResponse) CheckTimeout() bool {
	if time.Since(t.Timeout) > 0 {
		return true
	}

	return false
}
