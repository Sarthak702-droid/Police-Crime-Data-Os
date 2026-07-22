package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const maxAIResponseBytes = 4 << 20

type GeminiClient struct {
	baseURL string
	model   string
	apiKey  string
	http    *http.Client
}

func NewGeminiClient(baseURL, model, apiKey string) *GeminiClient {
	return &GeminiClient{baseURL: strings.TrimRight(baseURL, "/"), model: model, apiKey: apiKey, http: &http.Client{Timeout: 45 * time.Second}}
}

type GeminiFunctionCall struct {
	Name string
	ID   string
	Args map[string]interface{}
}

type GeminiTurn struct {
	Content      map[string]interface{}
	Text         string
	FunctionCall *GeminiFunctionCall
}

func (c *GeminiClient) Generate(ctx context.Context, systemInstruction string, contents []interface{}, tools []AIToolDefinition) (*GeminiTurn, error) {
	declarations := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		if tool.Implementation != "available" {
			continue
		}
		declarations = append(declarations, map[string]interface{}{"name": tool.Name, "description": tool.Description, "parameters": tool.InputSchema})
	}
	payload := map[string]interface{}{
		"systemInstruction": map[string]interface{}{"parts": []map[string]string{{"text": systemInstruction}}},
		"contents":          contents,
		"tools":             []map[string]interface{}{{"functionDeclarations": declarations}},
		"toolConfig":        map[string]interface{}{"functionCallingConfig": map[string]interface{}{"mode": "AUTO"}},
		"generationConfig":  map[string]interface{}{"temperature": 0.1, "maxOutputTokens": 2048},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxAIResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini returned HTTP %d", resp.StatusCode)
	}
	var decoded struct {
		Candidates []struct {
			Content map[string]interface{} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("invalid gemini response: %w", err)
	}
	if len(decoded.Candidates) == 0 {
		return nil, errors.New("gemini returned no candidate")
	}
	turn := &GeminiTurn{Content: decoded.Candidates[0].Content}
	parts, _ := turn.Content["parts"].([]interface{})
	for _, rawPart := range parts {
		part, _ := rawPart.(map[string]interface{})
		if text, ok := part["text"].(string); ok {
			turn.Text += text
		}
		if rawCall, ok := part["functionCall"].(map[string]interface{}); ok {
			call := &GeminiFunctionCall{}
			call.Name, _ = rawCall["name"].(string)
			call.ID, _ = rawCall["id"].(string)
			call.Args, _ = rawCall["args"].(map[string]interface{})
			if call.Args == nil {
				call.Args = map[string]interface{}{}
			}
			turn.FunctionCall = call
			break
		}
	}
	return turn, nil
}

type SarvamClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewSarvamClient(baseURL, apiKey string) *SarvamClient {
	return &SarvamClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, http: &http.Client{Timeout: 30 * time.Second}}
}

type TranslationResult struct {
	RequestID          string `json:"request_id"`
	TranslatedText     string `json:"translated_text"`
	SourceLanguageCode string `json:"source_language_code"`
	TargetLanguageCode string `json:"target_language_code"`
}

type TranscriptionResult struct {
	RequestID           string  `json:"request_id"`
	Transcript          string  `json:"transcript"`
	LanguageCode        string  `json:"language_code"`
	LanguageProbability float64 `json:"language_probability,omitempty"`
}

func (c *SarvamClient) Transcribe(ctx context.Context, filename string, audio []byte, language, mode string) (*TranscriptionResult, error) {
	if len(audio) == 0 {
		return nil, errors.New("audio file is required")
	}
	if language == "" {
		language = "unknown"
	}
	allowedModes := map[string]bool{"transcribe": true, "translate": true, "verbatim": true, "translit": true, "codemix": true}
	if !allowedModes[mode] {
		mode = "transcribe"
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(audio); err != nil {
		return nil, err
	}
	_ = writer.WriteField("model", "saaras:v3")
	_ = writer.WriteField("mode", mode)
	_ = writer.WriteField("language_code", language)
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/speech-to-text", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("api-subscription-key", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sarvam speech request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAIResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sarvam speech returned HTTP %d", resp.StatusCode)
	}
	var result TranscriptionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.Transcript) == "" {
		return nil, errors.New("sarvam returned an empty transcript")
	}
	return &result, nil
}

func (c *SarvamClient) Translate(ctx context.Context, input, sourceLanguage, targetLanguage string) (*TranslationResult, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("translation input is required")
	}
	if len([]rune(input)) > 1000 {
		return nil, errors.New("Mayura translation input cannot exceed 1000 characters")
	}
	if sourceLanguage == "" {
		sourceLanguage = "auto"
	}
	if targetLanguage != "kn-IN" && targetLanguage != "en-IN" {
		return nil, errors.New("target_language_code must be kn-IN or en-IN")
	}
	payload := map[string]interface{}{
		"input":                input,
		"source_language_code": sourceLanguage,
		"target_language_code": targetLanguage,
		"model":                "mayura:v1",
		"numerals_format":      "native",
		"mode":                 "modern-colloquial",
		"output_script":        "fully-native",
		"speaker_gender":       "Female",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/translate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-subscription-key", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sarvam request failed: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxAIResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sarvam returned HTTP %d", resp.StatusCode)
	}
	result := &TranslationResult{TargetLanguageCode: targetLanguage}
	if err := json.Unmarshal(responseBody, result); err != nil {
		return nil, fmt.Errorf("invalid sarvam response: %w", err)
	}
	if strings.TrimSpace(result.TranslatedText) == "" {
		return nil, errors.New("sarvam returned empty translated text")
	}
	return result, nil
}
