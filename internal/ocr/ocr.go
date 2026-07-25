package ocr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type OCRResult struct {
	ProductName string `json:"productName"`
	BatchNumber string `json:"batchNumber"`
	ExpiryDate  string `json:"expiryDate"`
	Model       string `json:"_model"`
	Error       string `json:"error,omitempty"`
}

func AnalyzeImages(b64Images []string, openRouterKey, openRouterModel, geminiKey string) *OCRResult {
	if openRouterKey != "" && openRouterModel != "" {
		result := analyzeOpenRouter(b64Images, openRouterKey, openRouterModel)
		if result.Error == "" {
			result.Model = "primary"
			return result
		}
	}
	if geminiKey != "" {
		result := analyzeGemini(b64Images, geminiKey)
		result.Model = "fallback"
		return result
	}
	return &OCRResult{Error: "No API keys configured"}
}

func analyzeOpenRouter(b64Images []string, apiKey, model string) *OCRResult {
	var content []map[string]any
	content = append(content, map[string]any{"type": "text", "text": "Analyze these product images and extract the details."})
	for _, b64 := range b64Images {
		content = append(content, map[string]any{
			"type": "image_url",
			"image_url": map[string]string{"url": "data:image/jpeg;base64," + b64},
		})
	}

	body := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{"role": "system", "content": ocrPrompt()},
			{"role": "user", "content": content},
		},
		"temperature":    0.1,
		"max_tokens":     256,
		"response_format": map[string]string{"type": "json_object"},
	}

	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "https://pims.local")
	req.Header.Set("X-Title", "PIMS Stock Take")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return parseOCRResponse(result)
}

func analyzeGemini(b64Images []string, apiKey string) *OCRResult {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=%s", apiKey)

	var parts []map[string]any
	parts = append(parts, map[string]any{"text": ocrPrompt()})
	for _, b64 := range b64Images {
		parts = append(parts, map[string]any{
			"inlineData": map[string]string{"mimeType": "image/jpeg", "data": b64},
		})
	}

	body := map[string]any{
		"contents": []map[string]any{{"parts": parts}},
		"generationConfig": map[string]any{
			"temperature":      0.1,
			"maxOutputTokens":  256,
			"responseMimeType": "application/json",
		},
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return parseOCRResponse(result)
}

func ocrPrompt() string {
	return `You are a pharmacy/lab inventory assistant. Look at these product images and extract the product name, batch number, and expiry date.

RULES:
- Product Name: Full prominent name visible.
- Batch Number: look for "Batch", "Lot", "LOT", "B/N", etc.
- Expiry Date: look for "EXP", "Expiry", "Use before", etc.
- Expiry MUST be in MM/YYYY format. If only a year is visible, use 12/YYYY.
- If a field is missing, use an empty string.
- Return ONLY valid JSON, no markdown.

Return EXACTLY: {"productName": "", "batchNumber": "", "expiryDate": ""}`
}

func parseOCRResponse(result map[string]any) *OCRResult {
	// OpenRouter format
	if choices, ok := result["choices"].([]any); ok && len(choices) > 0 {
		if msg, ok := choices[0].(map[string]any)["message"].(map[string]any); ok {
			if content, ok := msg["content"].(string); ok {
				var r OCRResult
				json.Unmarshal([]byte(content), &r)
				return &r
			}
		}
	}
	// Gemini format
	if candidates, ok := result["candidates"].([]any); ok && len(candidates) > 0 {
		if content, ok := candidates[0].(map[string]any)["content"].(map[string]any); ok {
			if parts, ok := content["parts"].([]any); ok && len(parts) > 0 {
				if text, ok := parts[0].(map[string]any)["text"].(string); ok {
					var r OCRResult
					json.Unmarshal([]byte(text), &r)
					return &r
				}
			}
		}
	}
	return &OCRResult{Error: "could not parse API response"}
}
