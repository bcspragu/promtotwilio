package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	twilio "github.com/twilio/twilio-go"
	twilioapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type MessageID string

// OptionsWithHandler is a struct with a mux and shared credentials
type Server struct {
	mux      *http.ServeMux
	sendText func(body string) (MessageID, error)
}

// NewMOptionsWithHandler returns a OptionsWithHandler for http requests
// with shared credentials
func NewServer(o *options) *Server {
	twilioC := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: o.AccountSid,
		Password: o.AuthToken,
	})
	s := &Server{
		sendText: func(body string) (MessageID, error) {
			params := &twilioapi.CreateMessageParams{
				To:   strPtr(o.Receiver),
				From: strPtr(o.Sender),
				Body: strPtr(body),
			}
			resp, err := twilioC.Api.CreateMessage(params)
			if err != nil {
				return "", fmt.Errorf("failed to send message: %w", err)
			}
			return MessageID(*resp.Sid), nil
		},
	}
	s.initMux()
	return s
}

func (s *Server) initMux() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.ping)
	mux.HandleFunc("/send", s.sendRequest)
	s.mux = mux
}

// HandleFastHTTP is the router function
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprint(w, "ping")
}

type reqBody struct {
	Status string
	Alerts []alert
}

type alert struct {
	Status      string
	Annotations map[string]string
	Labels      map[string]string
	StartsAt    time.Time
}

func (s *Server) sendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, http.StatusText(http.StatusNotAcceptable), http.StatusNotAcceptable)
		return
	}

	var req reqBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// TODO(brandon): Consider sending messages in this case as well, useful to
	// know if an alert resolves itself.
	if req.Status != "firing" {
		return
	}

	ctx := r.Context()
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()

	for i, alert := range req.Alerts {
		// We send all the messages sequentially with a basic rate limit to avoid
		// Twilio yelling at us in a severe outage.
		s.sendMessage(alert)
		if i == len(req.Alerts)-1 {
			// Last message, we don't need to wait.
			break
		}
		select {
		case <-t.C:
			// Continue
		case <-ctx.Done():
			log.Println("request context cancelled, not sending any more alerts for request")
			return
		}
	}
}

func (s *Server) sendMessage(alert alert) {
	body, err := findAndReplaceLabels(alert)
	if err != nil {
		log.Printf("failed to parse alert body: %v", err)
		return
	}
	body = "\"" + body + "\"" + " alert started at " + alert.StartsAt.Format(time.RFC1123)

	msgID, err := s.sendText(body)
	if err != nil {
		log.Printf("failed to send alert text message: %v", err)
		return
	}

	log.Printf("Sent message %q successfully", msgID)
}

func strPtr(in string) *string {
	return &in
}

func findAndReplaceLabels(alert alert) (string, error) {
	body := alert.Annotations["summary"]
	if body == "" {
		return "", errors.New("alert had no body")
	}

	labelReg := regexp.MustCompile(`\$labels\.[a-z]+`)
	matches := labelReg.FindAllString(body, -1)

	for _, match := range matches {
		labelName := strings.TrimPrefix(match, "$labels.")
		if labelName != "" {
			labelVal := alert.Labels[labelName]
			fmt.Println(labelName, labelVal)
			body = strings.Replace(body, match, labelVal, -1)
		}
	}

	return body, nil
}
