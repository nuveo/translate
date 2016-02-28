# Translate
Go online translation package

##### Install
`go get github.com/nuveo/translate`

## Available Translator API's
 * [Microsoft](http://www.microsoft.com/translator/getstarted.aspx)
 * or send us the next Translator API :smile:

## Usage

```go
package main

import (
  "fmt"
  "log"
  "github.com/nuveo/translate"
)

func main() {
  // gettind your credentials here in: http://www.microsoft.com/translator/getstarted.aspx
  t := &microsoft.AuthRequest{"client_id", "client_secret"}
  // Generate a token valid for 10 minutes
  tokenResponse := microsoft.GetAccessToken(t)

  text := "one two three"
  from := "en"
  to := "pt"

  toTranslate := &microsoft.TextTranslate{
	Text: text, From: from, To: to, TokenResponse: tokenResponse,
  }

  // Get translate of text, return the word translated and error
  resp, err := microsoft.TranslateText(toTranslate)
  if err != nil {
	log.Println(err)
  }
  fmt.Println(resp) // um dois três

  texts := []string{"one two three", "the book on the table"}
  toTranslate.Texts = texts

  // or get translate of array of strings
  resps, err := microsoft.TranslateTexts(toTranslate)
  if err != nil {
	log.Println(err)
  }
  fmt.Println(resps) // [um dois três o livro em cima da mesa]

  // Detect the language of texts
  detect := []string{"mundo", "world", "monde"}
  toTranslate.Texts = detect

  respsD, err := microsoft.DetectText(toTranslate)
  if err != nil {
	log.Println(err)
  }
  fmt.Println(respsD, toTranslate.Texts) // [es en fr]
}

```

### Support to cache with Redis

Now Translate make cache of words translated.

```go

  // to activate cache
  toTranslate := &microsoft.TextTranslate{
	Text: text, From: from, To: to, TokenResponse: tokenResponse, Cache: true,
  }

```
