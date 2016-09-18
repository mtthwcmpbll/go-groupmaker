// Based on some of the examples in the Google API Go Client Examples.
// https://github.com/google/google-api-go-client/blob/master/examples/

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	google "golang.org/x/oauth2/google"
	sheets "google.golang.org/api/sheets/v4"
)

var (
	clientID     = flag.String("clientid", "", "OAuth 2.0 Client ID.  If non-empty, overrides --clientid_file")
	secret       = flag.String("secret", "", "OAuth 2.0 Client Secret.  If non-empty, overrides --secret_file")
	cacheToken   = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
	debug        = flag.Bool("debug", false, "enable debug output")
	oauth2Config = &oauth2.Config{
		Endpoint:    google.Endpoint,
		Scopes:      []string{sheets.SpreadsheetsReadonlyScope},
		RedirectURL: "http://localhost:8080/oauth2Callback",
	}
	randState = fmt.Sprintf("st%d", time.Now().UnixNano())
)

func main() {
	flag.Parse()

	oauth2Config.ClientID = *clientID
	oauth2Config.ClientSecret = *secret

	// Set up the basic web services
	router := httprouter.New()
	router.GET("/", root)
	router.GET("/status", status)
	router.GET("/login", oauth2Login)
	router.GET("/oauth2Callback", oauth2Callback)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func root(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "TODO")
}

func status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprint(w, "OK!")
}

func oauth2Login(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	url := oauth2Config.AuthCodeURL(randState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func oauth2Callback(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	state := r.FormValue("state")
	if state != randState {
		log.Fatalf("invalid oauth state, expected '%s', got '%s'", randState, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := oauth2Config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Code exchange failed with '%v'", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// TODO: Do some stuff with the access

	ctx := context.Background()
	client := oauth2Config.Client(ctx, token)

	service, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to create Sheets service: %v", err)
	}

	spreadsheetId := "1L09WC9WB4kJTz3MkNXrc9MvtlplHKpxAhmb9PJ00LoE"
	spreadsheetRange := "Form Responses!A:Z"
	request := service.Spreadsheets.Values.Get(spreadsheetId, spreadsheetRange)

	response, err := request.Do()
	if err != nil {
		log.Fatalf("Unable to get sheet values: %v", err)
	}

	values := response.Values
	if values != nil && len(values) > 0 {
		for _, row := range values {
			// Print columns A and E, which correspond to indices 0 and 4.
			fmt.Println("{0}, {1}", row[0], row[2])
		}
	} else {
		fmt.Println("No data found.")
	}

	fmt.Fprint(w, "DONE!")
}
