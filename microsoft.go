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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nuveo/utils/cache"
)

const (
	datamarket        = "https://datamarket.accesscontrol.windows.net/v2/OAuth2-13"
	scope             = "http://api.microsofttranslator.com"
	translateURL      = "http://api.microsofttranslator.com/v2/Http.svc/Translate"
	translateArrayURL = "http://api.microsofttranslator.com/V2/Http.svc/TranslateArray"
	detectArrayURL    = "http://api.microsofttranslator.com/V2/Http.svc/DetectArray"
	grantType         = "client_credentials"
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
	templateToTranslate    = `<string xmlns="http://schemas.microsoft.com/2003/10/Serialization/Arrays">%s</string>`
	xmlDetectArrayTemplate = `<ArrayOfstring xmlns="http://schemas.microsoft.com/2003/10/Serialization/Arrays">%s</ArrayOfstring>`
)

type Access interface {
	GetAccessToken() TokenResponse
}

type Translator interface {
	Translate() (string, error)
	TranslateArray() ([]string, error)
	DetectTextArray() ([]string, error)
	CheckTimeout() bool
}

type AuthRequest struct {
	ClientID     string
	ClientSecret string
}

type TextTranslate struct {
	Text  string
	Texts []string
	From  string
	To    string
	Cache bool

	TokenResponse
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
	if t.CheckTimeout() == false {
		return t.Translate()
	}
	return "", errors.New("Access token is invalid, please get new token")
}

func TranslateTexts(t Translator) ([]string, error) {
	if t.CheckTimeout() == false {
		return t.TranslateArray()
	}
	return []string{}, errors.New("Access token is invalid, please get new token")
}

func DetectText(t Translator) ([]string, error) {
	if t.CheckTimeout() == false {
		return t.DetectTextArray()
	}
	return []string{}, errors.New("Access token is invalid, please get new token")
}

func rdbCache() *redis.Redis {
	redisUri := os.Getenv("REDIS")
	if redisUri == "" {
		redisUri = ":6739"
	}
	conn := redis.Connection{"tcp", redisUri, "7"}
	rdb, err := conn.Dial()
	if err != nil {
		log.Println("Fail to connect: ", err)
	}
	return rdb
}

// Make a POST request to `datamark` url for getting access token
func (a *AuthRequest) GetAccessToken() TokenResponse {
	client := &http.Client{}

	postValues := url.Values{}
	postValues.Add("client_id", a.ClientID)
	postValues.Add("client_secret", a.ClientSecret)
	postValues.Add("scope", scope)
	postValues.Add("grant_type", grantType)

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
	expiresIn, err := strconv.ParseInt(tr.ExpiresIn, 10, 0)
	if err != nil {
		log.Println(err)
	}

	expTime := now.Add(time.Duration(expiresIn) * time.Second)
	// 10 min
	tr.Timeout = expTime
	return tr
}

// Return `t.Text` in `t.From` language translated for `t.To` language
func (t *TextTranslate) Translate() (string, error) {

	if t.Cache == true {
		rdb := rdbCache()
		exists, _ := rdb.HExists(t.Text, t.To)
		if exists == true {
			result, _ := rdb.HGet(t.Text, t.To)
			log.Printf("Getting from cache %s:%s", t.Text, result)
			rdb.Conn.Close()
			return result, nil
		}
	}

	textEncode := url.Values{}
	textEncode.Add("text", t.Text)
	text := textEncode.Encode()

	url := fmt.Sprintf("%s?%s&from=%s&to=%s", translateURL, text, t.From, t.To)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	authToken := fmt.Sprintf("Bearer %s", t.TokenResponse.AccessToken)
	req.Header.Add("Authorization", authToken)

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
	if t.Cache == true {
		rdb := rdbCache()
		rdb.HSet(t.Text, t.To, obj.T)
		log.Printf("Add to cache %s:[%s] -> %s", t.Text, t.To, obj.T)
		rdb.Conn.Close()
	}

	return obj.T, nil
}

// Return `t.Texts` array in `t.From` language translated for `t.To` language
func (t *TextTranslate) TranslateArray() ([]string, error) {
	toTranslate := make([]string, len(t.Texts))
	response := []string{}

	// Simulate possible indexes of array response from microsoft.
	notCached := make(map[string]int)
	count := 0

	if t.Cache == true {
		rdb := rdbCache()
		for _, tx := range t.Texts {
			exs, _ := rdb.HExists(tx, t.To)

			if exs == false {
				notCached[tx] = count
				ts := fmt.Sprintf(templateToTranslate, tx)
				toTranslate = append(toTranslate, ts)
				count++
			}
		}
		rdb.Conn.Close()
	} else {
		for _, text := range t.Texts {
			tx := fmt.Sprintf(templateToTranslate, text)
			toTranslate = append(toTranslate, tx)
		}
	}

	// If do not need to do translation
	if len(notCached) == 0 {
		rdb := rdbCache()
		for _, tx := range t.Texts {
			res, _ := rdb.HGet(tx, t.To)
			log.Printf("Get from cache %s: [%s] %s", tx, t.To, res)
			if res != "" {
				response = append(response, res)
			}
		}
		rdb.Conn.Close()
		return response, nil
	}

	textToTranslate := strings.Join(toTranslate, "\n")
	bodyReq := fmt.Sprintf(xmlArrayTemplate, t.From, textToTranslate, t.To)

	client := &http.Client{}
	req, err := http.NewRequest("POST", translateArrayURL, strings.NewReader(bodyReq))
	if err != nil {
		log.Println(err)
	}

	authToken := fmt.Sprintf("Bearer %s", t.TokenResponse.AccessToken)
	req.Header.Add("Authorization", authToken)
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

	if t.Cache == true && len(obj.Resp) > 0 {
		rdb := rdbCache()
		for key, indx := range notCached {
			tx := obj.Resp[indx].Text
			log.Printf("Add to cache %s: [%s] %s", key, t.To, tx)

			err = rdb.HSet(key, t.To, tx)
			if err != nil {
				log.Println(err)
			}
		}

		for _, tx := range t.Texts {
			res, _ := rdb.HGet(tx, t.To)
			log.Printf("Get from cache %s: [%s] %s", tx, t.To, res)
			if res != "" {
				response = append(response, res)
			}
		}
		rdb.Conn.Close()
	}
	return response, nil
}

// Detects the language of the text passed in the array `t.Texts`
func (t *TextTranslate) DetectTextArray() ([]string, error) {
	response := []string{}
	toTranslate := make([]string, len(t.Texts))

	for _, text := range t.Texts {
		textEncode := url.Values{}
		textEncode.Add("text", text)
		text := textEncode.Encode()

		tx := fmt.Sprintf(templateToTranslate, text)
		toTranslate = append(toTranslate, tx)
	}
	textToTranslate := strings.Join(toTranslate, "\n")
	bodyReq := fmt.Sprintf(xmlDetectArrayTemplate, textToTranslate)

	client := &http.Client{}
	req, err := http.NewRequest("POST", detectArrayURL, strings.NewReader(bodyReq))
	if err != nil {
		log.Println(err)
	}

	authToken := fmt.Sprintf("Bearer %s", t.TokenResponse.AccessToken)
	req.Header.Add("Authorization", authToken)
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
	type LangArray struct {
		Array []string `xml:"string"`
	}

	var obj LangArray
	err = xml.Unmarshal(body, &obj)
	if err != nil {
		log.Println(err)
	}

	response = append(response, obj.Array...)
	return response, nil
}

// Verify if access token is valid
func (t *TokenResponse) CheckTimeout() bool {
	if time.Since(t.Timeout) > 0 {
		return true
	}
	return false
}
