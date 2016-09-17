// Based on some of the examples in the Google API Go Client Examples.
// https://github.com/google/google-api-go-client/blob/master/examples/

package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	google "golang.org/x/oauth2/google"
	sheets "google.golang.org/api/sheets/v4"
)

var (
	clientID   = flag.String("clientid", "", "OAuth 2.0 Client ID.  If non-empty, overrides --clientid_file")
	secret     = flag.String("secret", "", "OAuth 2.0 Client Secret.  If non-empty, overrides --secret_file")
	cacheToken = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
	debug      = flag.Bool("debug", false, "enable debug output")
)

func main() {
	flag.Parse()

	// Set up the basic web services
	router := httprouter.New()
	router.GET("/oauth", oauth2Callback)
	log.Fatal(http.ListenAndServe(":8080", router))

	config := &oauth2.Config{
		ClientID:     *clientID,
		ClientSecret: *secret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{sheets.SpreadsheetsReadonlyScope},
	}

	ctx := context.Background()
	client := newOAuthClient(ctx, config)

	service, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to create Sheets service: %v", err)
	}

	spreadsheetId := "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
	spreadsheetRange := "Class Data!A2:E"
	request := service.Spreadsheets.Values.Get(spreadsheetId, spreadsheetRange)

	response, err := request.Do()
	if err != nil {
		log.Fatalf("Unable to get sheet values: %v", err)
	}

	values := response.Values
	if values != nil && len(values) > 0 {
		fmt.Println("Name, Major")
		for _, row := range values {
			// Print columns A and E, which correspond to indices 0 and 4.
			fmt.Println("{0}, {1}", row[0], row[4])
		}
	} else {
		fmt.Println("No data found.")
	}
}

func newOAuthClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := tokenCacheFile(config)
	token, err := tokenFromFile(cacheFile)
	if err != nil {
		token = tokenFromWeb(ctx, config)
		saveToken(cacheFile, token)
	} else {
		log.Printf("Using cached token %#v from %q", token, cacheFile)
	}

	return config.Client(ctx, token)
}

func tokenCacheFile(config *oauth2.Config) string {
	hash := fnv.New32a()
	hash.Write([]byte(config.ClientID))
	hash.Write([]byte(config.ClientSecret))
	hash.Write([]byte(strings.Join(config.Scopes, " ")))
	fn := fmt.Sprintf("go-api-demo-tok%v", hash.Sum32())
	return filepath.Join(osUserCacheDir(), url.QueryEscape(fn))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	if !*cacheToken {
		return nil, errors.New("--cachetoken is false")
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := new(oauth2.Token)
	err = gob.NewDecoder(f).Decode(t)
	return t, err
}

func saveToken(file string, token *oauth2.Token) {
	f, err := os.Create(file)
	if err != nil {
		log.Printf("Warning: failed to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(token)
}

func tokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

	}))
	defer ts.Close()

	config.RedirectURL = "http://localhost:8080/oauth"
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Token exchange error: %v", err)
	}
	return token
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser.")
}

func osUserCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	case "linux", "freebsd":
		return filepath.Join(os.Getenv("HOME"), ".cache")
	}
	log.Printf("TODO: osUserCacheDir on GOOS %q", runtime.GOOS)
	return "."
}

func oauth2Callback(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if r.URL.Path == "/favicon.ico" {
		http.Error(w, "", 404)
		return
	}
	// if r.FormValue("state") != randState {
	// 	log.Printf("State doesn't match: req = %#v", r)
	// 	http.Error(w, "", 500)
	// 	return
	// }
	if code := r.FormValue("code"); code != "" {
		fmt.Fprintf(w, "<h1>Success</h1>Authorized.")
		w.(http.Flusher).Flush()
		//TODO: ch <- code
		return
	}
	log.Printf("no code")
	http.Error(w, "", 500)
}
