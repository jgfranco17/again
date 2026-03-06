package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jgfranco17/again"
)

// APIResponse represents a typical API response
type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Example 1: Retry HTTP requests with transient error handling
func retryHTTPRequest() {
	fmt.Println("=== Example 1: Retry HTTP Request ===")

	// Create a test server that fails a few times then succeeds
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		log.Printf("Server received request #%d", count)

		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Service temporarily unavailable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "success",
			Message: "Request processed successfully",
			Data:    map[string]string{"id": "12345"},
		})
	}))
	defer server.Close()

	// Configure retry with exponential backoff
	cfg := again.Config{
		Attempts: 5,
		Backoff:  again.Exponential(100 * time.Millisecond),
		Jitter:   again.FullJitter(),
		RetryIf:  again.TransientErrors,
		OnRetry: func(attempt int, err error) {
			log.Printf("Retrying HTTP request (attempt %d): %v", attempt, err)
		},
	}

	var response APIResponse
	err := again.Do(context.Background(), cfg, func() error {
		resp, err := http.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		// Retry on 5xx errors
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}

		// Don't retry on 4xx errors
		if resp.StatusCode >= 400 {
			return fmt.Errorf("client error: %d (won't retry)", resp.StatusCode)
		}

		return json.Unmarshal(body, &response)
	})

	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	log.Printf("Success! Response: %s - %s", response.Status, response.Message)
	fmt.Println()
}

// Example 2: API client with rate limiting and retry
func rateLimitedAPI() {
	fmt.Println("=== Example 2: Rate-Limited API Client ===")

	// Simulate a rate-limited API
	var requestCount atomic.Int32
	var lastRequestTime atomic.Value
	lastRequestTime.Store(time.Now())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		now := time.Now()
		lastTime := lastRequestTime.Load().(time.Time)

		// Simulate rate limit: max 1 request per 500ms
		if now.Sub(lastTime) < 500*time.Millisecond && count > 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Retry-After", "1")
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Rate limit exceeded",
			})
			log.Printf("Request #%d: Rate limit exceeded", count)
			return
		}

		lastRequestTime.Store(now)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Status: "success",
			Data:   map[string]interface{}{"request_id": count},
		})
		log.Printf("Request #%d: Success", count)
	}))
	defer server.Close()

	// Configure retry for rate limiting
	cfg := again.Config{
		Attempts: 6,
		Backoff:  again.Linear(600 * time.Millisecond), // Respect rate limit
		RetryIf: func(err error) bool {
			// Retry on rate limit errors
			return strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit")
		},
		OnRetry: func(attempt int, err error) {
			log.Printf("Rate limited, waiting before retry %d", attempt)
		},
	}

	start := time.Now()
	response, err := again.DoWithValue(context.Background(), cfg, func() (APIResponse, error) {
		resp, err := http.Get(server.URL)
		if err != nil {
			return APIResponse{}, err
		}
		defer resp.Body.Close()

		var apiResp APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return APIResponse{}, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			return APIResponse{}, fmt.Errorf("rate limit error: 429")
		}

		return apiResp, nil
	})

	elapsed := time.Since(start)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	log.Printf("Success after %v! Response: %+v", elapsed.Round(time.Millisecond), response)
	fmt.Println()
}

// Example 3: Multiple API endpoints with circuit breaker pattern
func multipleEndpoints() {
	fmt.Println("=== Example 3: Multiple Endpoints with Fallback ===")

	// Primary endpoint (fails)
	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Primary endpoint: failing")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer primaryServer.Close()

	// Backup endpoint (succeeds)
	backupServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Backup endpoint: success")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "success",
			Message: "Served from backup",
		})
	}))
	defer backupServer.Close()

	endpoints := []string{primaryServer.URL, backupServer.URL}

	cfg := again.Config{
		Attempts: len(endpoints),
		Backoff:  again.Constant(100 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			log.Printf("Trying endpoint %d", attempt+1)
		},
	}

	endpointIndex := 0
	response, err := again.DoWithValue(context.Background(), cfg, func() (APIResponse, error) {
		url := endpoints[endpointIndex]
		if endpointIndex < len(endpoints)-1 {
			endpointIndex++
		}

		resp, err := http.Get(url)
		if err != nil {
			return APIResponse{}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return APIResponse{}, fmt.Errorf("endpoint returned %d", resp.StatusCode)
		}

		var apiResp APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return APIResponse{}, err
		}

		return apiResp, nil
	})

	if err != nil {
		log.Fatalf("All endpoints failed: %v", err)
	}

	log.Printf("Success! %s", response.Message)
	fmt.Println()
}

// Example 4: POST request with idempotency
func idempotentPOST() {
	fmt.Println("=== Example 4: Idempotent POST Request ===")

	// Server that accepts idempotency key
	var processedKeys = make(map[string]bool)
	var attemptCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idempotencyKey := r.Header.Get("Idempotency-Key")
		attempt := attemptCount.Add(1)

		log.Printf("Received POST request (attempt %d) with key: %s", attempt, idempotencyKey)

		// Check if already processed
		if processedKeys[idempotencyKey] {
			log.Println("Request already processed (idempotent)")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "success",
				Message: "Already processed",
				Data:    map[string]string{"order_id": "ORD-12345"},
			})
			return
		}

		// Simulate failure on first attempt
		if attempt == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Temporary processing error",
			})
			return
		}

		// Process successfully
		processedKeys[idempotencyKey] = true
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "success",
			Message: "Order created",
			Data:    map[string]string{"order_id": "ORD-12345"},
		})
	}))
	defer server.Close()

	cfg := again.Config{
		Attempts: 3,
		Backoff:  again.Exponential(200 * time.Millisecond),
		Jitter:   again.EqualJitter(),
		OnRetry: func(attempt int, err error) {
			log.Printf("Retrying POST request (attempt %d)", attempt)
		},
	}

	// Create order with idempotency key
	idempotencyKey := fmt.Sprintf("order-%d", time.Now().Unix())
	payload := map[string]interface{}{
		"item":     "Widget",
		"quantity": 5,
	}
	payloadBytes, _ := json.Marshal(payload)

	response, err := again.DoWithValue(context.Background(), cfg, func() (APIResponse, error) {
		req, err := http.NewRequest("POST", server.URL+"/orders", strings.NewReader(string(payloadBytes)))
		if err != nil {
			return APIResponse{}, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return APIResponse{}, err
		}
		defer resp.Body.Close()

		var apiResp APIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return APIResponse{}, err
		}

		if resp.StatusCode >= 500 {
			return APIResponse{}, fmt.Errorf("server error: %d - %s", resp.StatusCode, apiResp.Message)
		}

		return apiResp, nil
	})

	if err != nil {
		log.Fatalf("Failed: %v", err)
	}

	log.Printf("Success! %s", response.Message)
	fmt.Println()
}

// Example 5: Context timeout with HTTP client
func contextTimeoutHTTP() {
	fmt.Println("=== Example 5: Context Timeout with HTTP ===")

	// Slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Slow server processing...")
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := again.Config{
		Attempts: 5,
		Backoff:  again.Constant(100 * time.Millisecond),
		OnRetry: func(attempt int, err error) {
			log.Printf("Request timed out, retry %d", attempt)
		},
	}

	// Set timeout that's shorter than server response time
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 500 * time.Millisecond, // Client timeout
	}

	err := again.Do(ctx, cfg, func() error {
		resp, err := client.Get(server.URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Context timeout exceeded: %v", err)
		} else if again.IsRetryError(err) {
			log.Printf("Retries exhausted: %v", err)
		} else {
			log.Printf("Request failed: %v", err)
		}
	}

	fmt.Println()
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	fmt.Println("Go Retry Framework - HTTP Client Examples")
	fmt.Println("==========================================")
	fmt.Println()

	retryHTTPRequest()
	rateLimitedAPI()
	multipleEndpoints()
	idempotentPOST()
	contextTimeoutHTTP()

	fmt.Println("All HTTP examples completed!")
}
