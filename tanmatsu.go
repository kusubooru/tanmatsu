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
	"time"
)

type Image struct {
	ID      int64
	URL     string
	Done    bool
	Kuid    int
	Created time.Time
	Player  string
	Channel string
}

type imageStore interface {
	GetImages(ctx context.Context, limit, offset int, orderAsc, isDone bool) ([]*Image, error)
}

type code string

const (
	internal code = "internal"
)

type Error struct {
	Code    code
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

type handler func(http.ResponseWriter, *http.Request) *Error

func (fn handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		//w.WriteHeader(e.Code)
		if e.Code == internal {
			log.Println(e.Err)
		}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type server struct {
	images   imageStore
	handlers http.Handler
}

func NewServer(imageStore imageStore) *server {
	s := &server{images: imageStore}

	mux := http.NewServeMux()
	mux.Handle("/images", handler(s.handleImages))
	s.handlers = mux
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handlers.ServeHTTP(w, r)
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
