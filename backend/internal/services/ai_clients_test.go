package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSarvamTranslateClient(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/translate" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("api-subscription-key") != "test-key" {
			t.Error("missing subscription key header")
		}
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["model"] != "mayura:v1" || request["target_language_code"] != "kn-IN" {
			t.Errorf("unexpected request: %#v", request)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"request_id":"r1","translated_text":"ನಮಸ್ಕಾರ","source_language_code":"en-IN"}`))}, nil
	})
	client := NewSarvamClient("https://api.sarvam.test", "test-key")
	client.http.Transport = transport
	result, err := client.Translate(context.Background(), "Hello", "en-IN", "kn-IN")
	if err != nil {
		t.Fatal(err)
	}
	if result.TranslatedText != "ನಮಸ್ಕಾರ" || result.TargetLanguageCode != "kn-IN" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGeminiFunctionCallParsing(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/models/gemini-3.5-flash:generateContent") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Error("missing Gemini key header")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"lookup_case","args":{"case_id":42},"id":"call-1"},"thoughtSignature":"opaque"}]}}]}`))}, nil
	})
	client := NewGeminiClient("https://gemini.test/v1beta", "gemini-3.5-flash", "test-key")
	client.http.Transport = transport
	turn, err := client.Generate(context.Background(), "system", []interface{}{map[string]interface{}{"role": "user", "parts": []map[string]string{{"text": "case 42"}}}}, AvailableAITools())
	if err != nil {
		t.Fatal(err)
	}
	if turn.FunctionCall == nil || turn.FunctionCall.Name != "lookup_case" || turn.FunctionCall.ID != "call-1" {
		t.Fatalf("unexpected function call: %#v", turn.FunctionCall)
	}
	if turn.FunctionCall.Args["case_id"] != float64(42) {
		t.Fatalf("unexpected args: %#v", turn.FunctionCall.Args)
	}
}

func TestSarvamSpeechToTextClient(t *testing.T) {
	client := NewSarvamClient("https://api.sarvam.test", "test-key")
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/speech-to-text" {
			t.Errorf("unexpected path %s", request.URL.Path)
		}
		if !strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Error("expected multipart request")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"request_id":"stt-1","transcript":"ನಮಸ್ಕಾರ","language_code":"kn-IN","language_probability":0.98}`))}, nil
	})
	result, err := client.Transcribe(context.Background(), "sample.wav", []byte("fake-audio"), "kn-IN", "transcribe")
	if err != nil {
		t.Fatal(err)
	}
	if result.Transcript != "ನಮಸ್ಕಾರ" || result.LanguageCode != "kn-IN" {
		t.Fatalf("unexpected result %#v", result)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }
