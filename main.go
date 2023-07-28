package main

import (
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	jose "gopkg.in/square/go-jose.v2"
)

//go:embed site/index.html
var indexHTML string

//go:embed site/success.html
var successHTML string

//go:embed site/unauthorized.html
var unauthorizedHTML string

//go:embed site/logo.png
var logoPNG []byte

type config struct {
	// port to listen on
	port int

	// Monocle settings
	privateKey    string
	token         string
	parsedPrivKey interface{}
	strictness    int

	// Fake user credentials
	username string
	password string
}

func main() {
	conf := parseConfigFromEnv()

	// Parse the private key
	privBytes, err := base64.StdEncoding.DecodeString(conf.privateKey)
	if err != nil {
		log.Fatalf("Error decoding private key: %v", err)
	}

	// Decode private key PEM string
	privPem, _ := pem.Decode(privBytes)

	// Parse private key bytes
	parsedKey, err := x509.ParsePKCS8PrivateKey(privPem.Bytes)
	if err != nil {
		log.Fatalf("Error parsing private key: %v", err)
	}

	conf.parsedPrivKey = parsedKey

	// Create router
	r := mux.NewRouter()

	// Index page
	r.HandleFunc("/", handleIndex(conf)).Methods("GET")

	// Login page
	r.HandleFunc("/login", handleUsernamePasswordFormPost(conf)).Methods("POST")

	// Serve the logo
	r.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(logoPNG) //nolint
	})

	log.Println("Listening on port", conf.port)

	// Run server
	log.Printf("Starting server with port %d, and token %s\n", conf.port, conf.token)
	err = http.ListenAndServe(fmt.Sprintf(":%d", conf.port), r)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

// indexPageData is the data for the index page, it contains just the token we need to inject into the template
type indexPageData struct {
	Token string
}

// handleIndex handles the index page
func handleIndex(c config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Serving index page %s for token %s\n", r.RequestURI, c.token)

		// index page template with token
		t, err := template.New("index").Parse(indexHTML)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		t.Execute(w, indexPageData{Token: c.token}) //nolint
	}
}

// usernamePasswordForm is the form data from the username/password form
type usernamePasswordForm struct {
	Username string
	Password string
	Bundle   string
}

// MonocleBundle is the bundle sent by Monocle
type MonocleBundle struct {
	VPN      bool   `json:"vpn"`
	Proxied  bool   `json:"proxied"`
	Anon     bool   `json:"anon"`
	IP       string `json:"ip"`
	TS       string `json:"ts"`
	Complete bool   `json:"complete"`
	ID       string `json:"id"`
	SID      string `json:"sid"`
}

type unauthorizedPageData struct {
	Reason string
}

// handleUsernamePasswordFormPost handles the username/password form post
func handleUsernamePasswordFormPost(conf config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling username/password form post")

		r.ParseForm() //nolint
		username := r.Form.Get("username")
		password := r.Form.Get("password")
		monocleBundle := r.Form.Get("monocle")

		log.Printf("Recieved post for username %s with bundle %s", username, monocleBundle)

		// Parse the encrypted Monocle bundle
		jwe, err := jose.ParseEncrypted(monocleBundle)
		if err != nil {
			fmt.Println("Error parsing encrypted Monocle bundle")
			return
		}

		// Decrypt the bundle with the private key
		decryptedBundle, err := jwe.Decrypt(conf.parsedPrivKey)
		if err != nil {
			log.Printf("Error decrypting Monocle bundle: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("Decrypted Monocle bundle:", string(decryptedBundle))

		// Parse the decrypted bundle as JSON
		var bundle MonocleBundle
		err = json.Unmarshal(decryptedBundle, &bundle)
		if err != nil {
			log.Printf("Error parsing decrypted Monocle bundle: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Analyze the bundle and determine if we should block the request
		shouldBlock, reason := block(conf, bundle)
		if shouldBlock {
			log.Printf("Blocking request for username %s with bundle %s", username, monocleBundle)
			w.WriteHeader(http.StatusUnauthorized)
			// index page template with token
			t, err := template.New("index").Parse(unauthorizedHTML)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			t.Execute(w, unauthorizedPageData{Reason: reason}) //nolint
			w.Write([]byte(unauthorizedHTML))                  //nolint
		}

		// If they provided the correct username and password, show the success page with the decrypted bundle for visual confirmation
		if username == conf.username && password == conf.password {
			log.Printf("Showing success page for username %s with bundle %s", username, monocleBundle)

			// Format the bundle JSON nicely
			decryptedBundle, err = json.MarshalIndent(bundle, "", "  ")
			if err != nil {
				log.Printf("Error marshalling decrypted bundle: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			t, err := template.New("success").Parse(successHTML)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			t.Execute(w, usernamePasswordForm{Username: username, Password: password, Bundle: string(decryptedBundle)}) //nolint

			return
		}

		// Otherwise, show the unauthorized page
		log.Printf("Showing unauthorized page for username %s with bundle %s", username, monocleBundle)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(unauthorizedHTML)) //nolint
	}
}

// parseConfigFromEnv parses the config from environment variables
func parseConfigFromEnv() config {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		log.Fatal("PORT environment variable not set")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid PORT value: %v", err)
	}

	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		log.Fatal("PRIVATE_KEY environment variable not set")
	}

	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN environment variable not set")
	}

	strictness := os.Getenv("STRICTNESS_LEVEL")
	if strictness == "" {
		log.Fatal("STRICTNESS_LEVEL environment variable not set")
	}

	strictnessInt, err := strconv.Atoi(strictness)
	if err != nil {
		log.Fatalf("invalid STRICTNESS_LEVEL value: %v", err)
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		log.Fatal("USERNAME environment variable not set")
	}

	password := os.Getenv("PASSWORD")
	if password == "" {
		log.Fatal("PASSWORD environment variable not set")
	}

	return config{
		port:       port,
		privateKey: privateKey,
		strictness: strictnessInt,
		token:      token,
		username:   username,
		password:   password,
	}
}

func block(cfg config, b MonocleBundle) (bool, string) {
	// If we couldn't complete the analysis, block
	if !b.Complete {
		log.Println("bundle not complete, blocking")
		return true, "bundle not complete"
	}

	// If the timestamp is empty, block
	if b.TS == "" {
		log.Println("bundle timestamp empty, blocking")
		return true, "bundle timestamp empty"
	}

	// If the timestamp is too old, block
	parsedTimestamp, err := time.Parse(time.RFC3339, b.TS)
	if err != nil {
		log.Printf("Error parsing timestamp: %v", err)
		return true, "bundle timestamp invalid"
	}

	if time.Since(parsedTimestamp) > time.Hour {
		log.Println("bundle timestamp too old, blocking")
		return true, "bundle timestamp too old"
	}

	// Depending on the strictness level, block if the bundle is a VPN, proxy, or anonymous
	switch cfg.strictness {
	case 0:
		// Log only
		log.Printf("strictness level log only, doing nothing for vpn: %v, proxied, %v, anon: %v", b.VPN, b.Proxied, b.Anon)
		return false, ""
	case 1:
		// Block proxies
		return b.Proxied && b.Anon, "no proxies allowed"
	case 2:
		// Block vpns
		return b.VPN && b.Anon, "no vpns allowed"
	case 3:
		// Block vpns and proxies
		return (b.VPN || b.Proxied) && b.Anon, "no vpns or proxies allowed"
	default:
		// Default to log only
		log.Printf("strictness level log only, doing nothing for vpn: %v, proxied, %v, anon: %v", b.VPN, b.Proxied, b.Anon)
		return false, ""
	}
}
