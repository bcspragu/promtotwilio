package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestSendRequest(t *testing.T) {
	var messages []string
	s := &Server{
		sendText: func(body string) (MessageID, error) {
			messages = append(messages, body)
			return "msgid-1", nil
		},
	}
	s.initMux()
	// Taken from https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
	body := `{
  "version": "4",
  "groupKey": "not-used",
  "truncatedAlerts": 0,
  "status": "firing",
  "receiver": "not-used",
  "groupLabels": {"not": "used"},
  "commonLabels": {"not": "used"},
  "commonAnnotations": {"not": "used"},
  "externalURL": "https://youralertmanager/the-alert",
  "alerts": [
    {
      "status": "firing",
      "labels": {
				"alertname": "InstanceDown",
				"instance":  "http://example.com",
				"job":       "blackbox"
			},
      "annotations": {
				"description": "Unable to scrape $labels.instance",
				"summary":     "Address $labels.instance appears to be down with $labels.alertname"
			},
      "startsAt": "2017-01-06T19:34:52.887Z",
      "endsAt": "0000-01-01T00:00:00.000Z",
      "generatorURL": "https://probably/dns/or/something",
      "fingerprint": "abc321"
    }
  ]
}`

	r := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// We send it here instead of directly to sendRequest() to make sure our routing is good.
	s.ServeHTTP(w, r)

	if statusCode := w.Result().StatusCode; statusCode != http.StatusOK {
		t.Fatalf("sendRequest returned non-200 code %d", statusCode)
	}

	wantMessages := []string{
		`"Address http://example.com appears to be down with InstanceDown" alert started at Fri, 06 Jan 2017 19:34:52 UTC`,
	}

	if diff := cmp.Diff(wantMessages, messages); diff != "" {
		t.Errorf("unexpected set of text messages sent (-want +got)\n%s", diff)
	}
}

func TestFindAndReplaceLabels(t *testing.T) {
	alert := alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "InstanceDown",
			"instance":  "http://example.com",
			"job":       "blackbox",
		},
		Annotations: map[string]string{
			"description": "Unable to scrape $labels.instance",
			"summary":     "Address $labels.instance appears to be down with $labels.alertname",
		},
		StartsAt: time.Date(2017, time.January, 6, 19, 34, 52, 887000000, time.UTC),
	}

	got, err := findAndReplaceLabels(alert)
	if err != nil {
		t.Fatalf("findAndReplaceLabels: %v", err)
	}
	want := "Address http://example.com appears to be down with InstanceDown"
	if got != want {
		t.Errorf("findAndReplaceLabels = %q, want %q", got, want)
	}
}
