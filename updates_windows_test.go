//go:build windows

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReleaseVersionOrdering(t *testing.T) {
	for _, test := range []struct {
		current, latest string
		order           int
		valid           bool
	}{
		{"1.9.0", "1.10.0", -1, true},
		{"2.0.0", "1.99.99", 1, true},
		{"1.2.0", "1.2.0", 0, true},
		{"1.2.0", "1.2.1", -1, true},
		{"dev", "1.2.0", 0, false},
		{"1.2.0", "1.3.0-beta", 0, false},
		{"1.02.0", "1.2.0", 0, false},
		{"1.0.0", "99999999999999999999.0.0", -1, true},
	} {
		order, valid := compareReleaseVersions(test.current, test.latest)
		if order != test.order || valid != test.valid {
			t.Fatalf("%s versus %s: %d, %t", test.current, test.latest, order, valid)
		}
	}
}

func TestLatestReleaseUsesOnlyOfficialStableRedirect(t *testing.T) {
	for _, test := range []struct {
		location string
		status   int
		want     string
	}{
		{"https://github.com/indiefun/simple-keymap/releases/tag/v1.10.0", 302, "1.10.0"},
		{"https://github.com/indiefun/simple-keymap/releases/tag/v1.2.0-beta", 302, ""},
		{"https://github.com/indiefun/other/releases/tag/v1.2.0", 302, ""},
		{"https://example.com/indiefun/simple-keymap/releases/tag/v1.2.0", 302, ""},
		{"http://github.com/indiefun/simple-keymap/releases/tag/v1.2.0", 302, ""},
		{"https://github.com/indiefun/simple-keymap/releases/tag/v01.2.0", 302, ""},
		{"https://github.com/indiefun/simple-keymap/releases/tag/v1.2.0?redirect=other", 302, ""},
		{"", 403, ""}, {"", 200, ""},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.UserAgent() != "SimpleKeymap/"+appVersion {
				t.Error("unexpected release request")
			}
			w.Header().Set("Location", test.location)
			w.WriteHeader(test.status)
		}))
		version, err := fetchLatestRelease(context.Background(), releaseClient(), server.URL)
		server.Close()
		if version != test.want || (err == nil) != (test.want != "") {
			t.Fatalf("%s: version=%s, err=%v", test.location, version, err)
		}
	}
}

func TestReleaseCheckCanBeCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if version, err := fetchLatestRelease(ctx, releaseClient(), server.URL); err == nil || version != "" {
		t.Fatal("cancelled check returned a version")
	}
}
