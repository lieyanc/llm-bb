package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"llm-bb/internal/config"
)

type Client struct {
	httpClient         *http.Client
	defaultTimeout     time.Duration
	defaultTemperature float64
	defaultMaxTokens   int
	requestRetries     int
	retryBackoff       time.Duration
	maxResponseBytes   int64
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	BaseURL     string
	APIKey      string
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

type ChatResponse struct {
	Content          string
	PromptTokens     int
	CompletionTokens int
}

func NewClient(defaultTimeout time.Duration) *Client {
	defaults := config.Default()
	return NewClientWithConfig(defaultTimeout, defaults.LLM)
}

func NewClientWithConfig(defaultTimeout time.Duration, cfg config.LLMConfig) *Client {
	if cfg.DefaultTemperature <= 0 {
		cfg.DefaultTemperature = 0.9
	}
	if cfg.DefaultMaxTokens <= 0 {
		cfg.DefaultMaxTokens = 256
	}
	if cfg.RequestRetries <= 0 {
		cfg.RequestRetries = 3
	}
	if cfg.RetryBackoffMS <= 0 {
		cfg.RetryBackoffMS = 350
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = 1 << 20
	}

	return &Client{
		httpClient:         &http.Client{},
		defaultTimeout:     defaultTimeout,
		defaultTemperature: cfg.DefaultTemperature,
		defaultMaxTokens:   cfg.DefaultMaxTokens,
		requestRetries:     cfg.RequestRetries,
		retryBackoff:       time.Duration(cfg.RetryBackoffMS) * time.Millisecond,
		maxResponseBytes:   cfg.MaxResponseBytes,
	}
}

func (c *Client) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(req.BaseURL) == "" {
		return ChatResponse{}, errors.New("base URL is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return ChatResponse{}, errors.New("model is required")
	}
	if len(req.Messages) == 0 {
		return ChatResponse{}, errors.New("messages are required")
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = c.defaultMaxTokens
	}
	if req.Temperature == 0 {
		req.Temperature = c.defaultTemperature
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload := map[string]any{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.requestRetries; attempt++ {
		response, err := c.doRequest(ctx, req, body)
		if err == nil {
			return response, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			break
		}

		select {
		case <-ctx.Done():
			return ChatResponse{}, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * c.retryBackoff):
		}
	}

	return ChatResponse{}, lastErr
}

func (c *Client) doRequest(ctx context.Context, req ChatRequest, body []byte) (ChatResponse, error) {
	url := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(req.APIKey); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, fmt.Errorf("upstream status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}

	if len(decoded.Choices) == 0 {
		return ChatResponse{}, errors.New("response contains no choices")
	}

	return ChatResponse{
		Content:          strings.TrimSpace(decoded.Choices[0].Message.Content),
		PromptTokens:     decoded.Usage.PromptTokens,
		CompletionTokens: decoded.Usage.CompletionTokens,
	}, nil
}
