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
	datamarket        = "https://datamarket.accesscontrol.windows.net/v2/OAuth2-13"
	scope             = "http://api.microsofttranslator.com"
	translateUrl      = "http://api.microsofttranslator.com/v2/Http.svc/Translate"
	translateArrayUrl = "http://api.microsofttranslator.com/V2/Http.svc/TranslateArray"
	grant_type        = "client_credentials"
	xmlArrayTemplate  = `<TranslateArrayRequest>
						<AppId />
						<From>%s</From>
						<Options>
							<Category xmlns="http://schemas.datacontract.org/2004/07/Microsoft.MT.Web.Service.V2" ></Category>
							<ContentType xmlns="http://schemas.datacontract.org/2004/07/Microsoft.MT.Web.Service.V2">text/plain</ContentType>
							<ReservedFlags xmlns="http://schemas.datacontract.org/2004/07/Microsoft.MT.Web.Service.V2" />
							<State xmlns="http://schemas.datacontract.org/2004/07/Microsoft.MT.Web.Service.V2"></State>
							<Uri xmlns="http://schemas.datacontract.org/2004/07/Microsoft.MT.Web.Service.V2"></Uri>
							<User xmlns="http://schemas.datacontract.org/2004/07/Microsoft.MT.Web.Service.V2"></User>
						</Options>
						<Texts>
							%s
						</Texts>
						<To>%s</To>
					</TranslateArrayRequest>`
	templateToTranslate = `<string xmlns="http://schemas.microsoft.com/2003/10/Serialization/Arrays">%s</string>`
)

type Access interface {
	GetAccessToken() TokenResponse
}

type Translator interface {
	Translate() (string, error)
	TranslateArray() ([]string, error)
}

type AuthRequest struct {
	ClientId     string
	ClientSecret string
}

type TextTranslate struct {
	Text          string
	Texts         []string
	From          string
	To            string
	TokenResponse TokenResponse
}

type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type,omitempty"`
	ExpiresIn   string    `json:"expires_in"`
	Scope       string    `json:"scope"`
	Timeout     time.Time `json:"timeout"`
}

type TranslateResponse struct {
	Resp []ArrayResp `xml:"TranslateArrayResponse"`
}

type ArrayResp struct {
	Text string `xml:"TranslatedText"`
}

func GetAccessToken(access Access) TokenResponse {
	return access.GetAccessToken()
}

func TranslateText(t Translator) (string, error) {
	return t.Translate()
}

func TranslateTexts(t Translator) ([]string, error) {
	return t.TranslateArray()
}

// Make a POST request to `datamark` url
func (a *AuthRequest) GetAccessToken() TokenResponse {
	client := &http.Client{}

	postValues := url.Values{}
	postValues.Add("client_id", a.ClientId)
	postValues.Add("client_secret", a.ClientSecret)
	postValues.Add("scope", scope)
	postValues.Add("grant_type", grant_type)

	req, err := http.NewRequest("POST", datamarket, strings.NewReader(postValues.Encode()))
	if err != nil {
		log.Println(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

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

// Return `text` in `from` language translated for `to` language
func (t *TextTranslate) Translate() (string, error) {

	if t.TokenResponse.CheckTimeout() == true {
		return "", errors.New("Access token is invalid, please get new token")
	}
	textEncode := url.Values{}
	textEncode.Add("text", t.Text)
	text := textEncode.Encode()

	url := fmt.Sprintf("%s?%s&from=%s&to=%s", translateUrl, text, t.From, t.To)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	auth_token := fmt.Sprintf("Bearer %s", t.TokenResponse.AccessToken)
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

// Return `texts` array in `from` language translated for `to` language
func (t *TextTranslate) TranslateArray() ([]string, error) {
	if t.TokenResponse.CheckTimeout() == true {
		return []string{}, errors.New("Access token is invalid, please get new token")
	}
	response := []string{}
	toTranslate := make([]string, len(t.Texts))

	for _, text := range t.Texts {
		t := fmt.Sprintf(templateToTranslate, text)
		toTranslate = append(toTranslate, t)
	}
	textToTranslate := strings.Join(toTranslate, "\n")
	bodyReq := fmt.Sprintf(xmlArrayTemplate, t.From, textToTranslate, t.To)

	client := &http.Client{}
	req, err := http.NewRequest("POST", translateArrayUrl, strings.NewReader(bodyReq))
	if err != nil {
		log.Println(err)
	}

	auth_token := fmt.Sprintf("Bearer %s", t.TokenResponse.AccessToken)
	req.Header.Add("Authorization", auth_token)
	req.Header.Add("Content-Type", "text/xml")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var obj TranslateResponse
	err = xml.Unmarshal(body, &obj)
	if err != nil {
		log.Println(err)
	}

	for _, t := range obj.Resp {
		response = append(response, t.Text)
	}

	return response, nil
}

// Verify if access token is valid
func (t *TokenResponse) CheckTimeout() bool {
	if time.Since(t.Timeout) > 0 {
		return true
	}
	return false
}
