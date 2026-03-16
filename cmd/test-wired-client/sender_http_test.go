package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPCardSender_SendCard(t *testing.T) {
	var gotReq PostCardRequest
	var gotContentType string
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewHTTPCardSender(srv.URL)
	err := sender.SendCard(context.Background(), "04AABBCCDD11", "device-01", 1)
	if err != nil {
		t.Fatalf("SendCard() error = %v", err)
	}

	if gotPath != "/card" {
		t.Errorf("path = %q, want /card", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotReq.UID != "04AABBCCDD11" {
		t.Errorf("UID = %q, want 04AABBCCDD11", gotReq.UID)
	}
	if gotReq.DeviceID != "device-01" {
		t.Errorf("DeviceID = %q, want device-01", gotReq.DeviceID)
	}
	if gotReq.PairID != 1 {
		t.Errorf("PairID = %d, want 1", gotReq.PairID)
	}
}

func TestHTTPCardSender_SendBoot(t *testing.T) {
	var gotDevice Device
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotDevice)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := NewHTTPCardSender(srv.URL)
	err := sender.SendBoot(context.Background(), "device-01", []int{1, 2})
	if err != nil {
		t.Fatalf("SendBoot() error = %v", err)
	}

	if gotPath != "/device/boot" {
		t.Errorf("path = %q, want /device/boot", gotPath)
	}
	if gotDevice.DeviceID != "device-01" {
		t.Errorf("DeviceID = %q, want device-01", gotDevice.DeviceID)
	}
	if len(gotDevice.PairIDs) != 2 || gotDevice.PairIDs[0] != 1 || gotDevice.PairIDs[1] != 2 {
		t.Errorf("PairIDs = %v, want [1, 2]", gotDevice.PairIDs)
	}
}

func TestHTTPCardSender_SendCard_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	sender := NewHTTPCardSender(srv.URL)
	err := sender.SendCard(context.Background(), "04AABBCCDD11", "device-01", 1)
	if err == nil {
		t.Fatal("SendCard() expected error for 500 response")
	}
}

func TestHTTPCardSender_SendBoot_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	sender := NewHTTPCardSender(srv.URL)
	err := sender.SendBoot(context.Background(), "device-01", []int{1})
	if err == nil {
		t.Fatal("SendBoot() expected error for 500 response")
	}
}

func TestHTTPCardSender_Mode(t *testing.T) {
	sender := NewHTTPCardSender("http://localhost:8080")
	if got := sender.Mode(); got != "HTTP" {
		t.Errorf("Mode() = %q, want HTTP", got)
	}
}
