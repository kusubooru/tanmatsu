package tanmatsu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type Image struct {
	ID      int64     `json:"id"`
	URL     string    `json:"url"`
	Done    bool      `json:"done"`
	Kuid    int       `json:"kuid"`
	Created time.Time `json:"created"`
	Player  string    `json:"player"`
	Channel string    `json:"channel"`
}

type imageStore interface {
	GetImages(ctx context.Context, limit, offset int, orderAsc, isDone bool) ([]*Image, error)
}

const (
	internal     = "internal"
	unauthorized = "unauthorized"
)

var toHTTPStatus = map[string]int{
	internal:     http.StatusInternalServerError,
	unauthorized: http.StatusUnauthorized,
}

func HTTPStatus(c string) int {
	status, ok := toHTTPStatus[c]
	if !ok {
		return http.StatusInternalServerError
	}
	return status
}

type Error struct {
	Code    string
	Message string
	Op      string
	Err     error
}

func (e *Error) Error() string {
	var buf bytes.Buffer

	// Print the current operation in our stack, if any.
	if e.Op != "" {
		fmt.Fprintf(&buf, "%s: ", e.Op)
	}

	// If wrapping an error, print its Error() message.
	// Otherwise print the error code & message.
	if e.Err != nil {
		buf.WriteString(e.Err.Error())
	} else {
		if e.Code != "" {
			fmt.Fprintf(&buf, "<%s> ", e.Code)
		}
		buf.WriteString(e.Message)
	}
	return buf.String()
}

// ErrorCode returns the code of the root error, if available. Otherwise returns internal.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	} else if e, ok := err.(*Error); ok && e.Code != "" {
		return e.Code
	} else if ok && e.Err != nil {
		return ErrorCode(e.Err)
	}
	return internal
}

type handler func(http.ResponseWriter, *http.Request) *Error

func (fn handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(HTTPStatus(ErrorCode(e)))
		if ErrorCode(e) == internal {
			log.Println(e.Err)
		}

		type ErrorResp struct {
			Message string `json:"message"`
			Err     string `json:"err"`
		}
		resp := ErrorResp{Message: e.Message, Err: e.Err.Error()}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type server struct {
	images   imageStore
	handlers http.Handler
	version  string
	issuer   string
	secret   []byte
}

func NewServer(imageStore imageStore, version, issuer string, secret []byte) *server {
	s := &server{
		images:  imageStore,
		version: version,
		issuer:  issuer,
		secret:  secret,
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler(s.handleHome))
	mux.Handle("/images", s.auth(s.handleImages))
	s.handlers = mux
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handlers.ServeHTTP(w, r)
}

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) *Error {
	fmt.Fprintln(w, s.version)
	return nil
}

func (s *server) auth(h handler) handler {
	var op = "auth"
	var msg = "Authorization failed."
	return func(w http.ResponseWriter, r *http.Request) *Error {
		token, err := getTokenFromHeader(r.Header)
		if err != nil {
			return &Error{Code: unauthorized, Op: op, Message: msg, Err: err}
		}

		if err := validateToken(token, s.secret, s.issuer); err != nil {
			return &Error{Code: unauthorized, Op: op, Message: msg, Err: err}
		}

		return h(w, r)
	}
}

func getTokenFromHeader(header map[string][]string) (string, error) {
	var tokenString string
	tokens, ok := header["Authorization"]
	if !ok || len(tokens) < 1 {
		return "", fmt.Errorf("expecting Authorization header")
	}

	tokenString = tokens[0]
	if !strings.HasPrefix(tokenString, "Bearer ") {
		return "", fmt.Errorf("expecting Bearer auth type")
	}
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	return tokenString, nil
}

func validateToken(tokenString string, secret []byte, issuer string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return secret, nil
	})
	if err != nil {
		return fmt.Errorf("parsing token: %v", err)
	}

	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("expecting MapClaims")
	}
	iss, ok := claims["iss"]
	if !ok {
		return fmt.Errorf("no issuer defined in claims")
	}
	if iss != issuer {
		return fmt.Errorf("unknown issuer")
	}
	return nil
}

func (s *server) handleImages(w http.ResponseWriter, r *http.Request) *Error {
	v := urlValues{r.URL.Query()}

	orderAsc := v.Bool("order", "asc")
	isDone := v.Bool("done", "true")
	limit := v.Int("limit", 10)
	offset := v.Int("offset", 0)

	images, err := s.images.GetImages(r.Context(), limit, offset, orderAsc, isDone)
	if err != nil {
		return &Error{Code: internal, Err: err, Message: "cannot get images"}
	}
	if err := json.NewEncoder(w).Encode(images); err != nil {
		return &Error{Code: internal, Err: err, Message: "cannot encode"}
	}
	return nil
}

type urlValues struct {
	url.Values
}

func (v urlValues) Bool(key, value string) bool {
	val := v.Get(key)
	return val == value
}

func (v urlValues) Int(key string, defaultValue int) int {
	a := v.Get(key)
	i, err := strconv.Atoi(a)
	if err != nil {
		return defaultValue
	}
	return i
}
