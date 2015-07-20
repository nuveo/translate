# Translate
Go online translation package

##### Install
`go get github.com/poorny/translate`

## Available Translator API's
 * [Microsoft](http://www.microsoft.com/translator/getstarted.aspx)
 * or send us the next Translator API :smile:

## Usage

```go
package main

import (
  "fmt"
  "log"
  "github.com/poorny/translate"
)

func main() {
  // gettind your credentials here in: http://www.microsoft.com/translator/getstarted.aspx
  t := microsoft.AuthRequest{"client_id", "client_secret"}
  // Generate a token valid for 10 minutes
  tokenResponse := t.GetAccessToken()

  text := "one two three"
  from := "en"
  to := "pt"
  // Get translate of text, return the word translated and error
  resp, err := tokenResponse.Translate(text, from, to)
  if err != nil {
    log.Println(err)
  }
  fmt.Println(resp) // um dois três

  // or get translate of array of strings
  texts := []string{"one two three", "the book on the table"}

  resp, err := tokenResponse.TranslateArray(texts, from, to)
  if err != nil {
    log.Println(err)
  }
  fmt.Println(resp) // um dois três, o livro em cima da mesa
}

```
