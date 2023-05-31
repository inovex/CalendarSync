package auth

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"golang.org/x/oauth2"
)

const successHtml = `<!DOCTYPE html>
<html>
<head>
    <title>CalendarSync</title>
</head>
<body style='font-family: "Helvetica Neue",Helvetica,Arial,sans-serif;'>
    <div style="text-align: center; padding-top: 30px;">
        <h2 style="color:#0fad00; font-weight: 500; font-size: 30px; margin-bottom: 10px;">CalendarSync authentication successful!</h2>
        <p style="font-size:20px; color:#5C5C5C; margin-top: 10px;">You can now close this window.</p>
    </div>
</body>
</html>`

type OAuthHandler struct {
	listener net.Listener
	config   oauth2.Config
	token    *oauth2.Token
}

func NewOAuthHandler(config oauth2.Config, bindPort uint) (*OAuthHandler, error) {
	address := net.JoinHostPort("localhost", strconv.Itoa(int(bindPort)))
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &OAuthHandler{
		config:   config,
		listener: listener,
	}, nil
}

func (l *OAuthHandler) Configuration() *oauth2.Config {
	redirectURL := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort("localhost", strconv.Itoa(l.listener.Addr().(*net.TCPAddr).Port)),
		Path:   "/redirect",
	}

	return &oauth2.Config{
		ClientID:     l.config.ClientID,
		ClientSecret: l.config.ClientSecret,
		Endpoint:     l.config.Endpoint,
		RedirectURL:  redirectURL.String(),
		Scopes:       l.config.Scopes,
	}
}

func (l *OAuthHandler) Token() *oauth2.Token {
	return l.token
}

func (l *OAuthHandler) createAuthorizationExchange(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		defer l.listener.Close()

		authorizationCode := req.URL.Query().Get("code")
		var err error

		// exchange authorization token for access and refresh token
		l.token, err = l.Configuration().Exchange(context.WithValue(ctx, oauth2.HTTPClient, http.DefaultClient), authorizationCode)
		if err != nil {
			log.Error(err, "method", "createAuthorizationExchange")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// show the user a success page and stop the http listener
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(successHtml)); err != nil {
			panic(err)
		}
	}
}

// Listen is meant to be called as goroutine. Once your handler has finished, just cancel the context
// and the http server will shut down.
func (l *OAuthHandler) Listen(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", l.createAuthorizationExchange(ctx))

	if err := http.Serve(l.listener, mux); err != nil {
		// Chrome sometimes requests the favicon after the oauth request and we do not want to panic here.
		if strings.Contains(err.Error(), "use of closed network connection") {
			return nil
		}
		return err
	}

	return nil
}
