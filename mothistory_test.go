package mothistory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func createMockServer() *httptest.Server {
	handler := http.NewServeMux()

	handler.HandleFunc("/registration/ML58FOU", func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{"registration": "ML58FOU"}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, mockResponse)
	})

	handler.HandleFunc("/vin/BNR32305366", func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{"vin": "BNR32305366"}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, mockResponse)
	})

	handler.HandleFunc("/bulk-download", func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{"bulk": [], "delta": []}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, mockResponse)
	})

	handler.HandleFunc("/credentials", func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `{"clientSecret": "your-new-client-secret-value"}`
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		w.Header().Set("X-API-Key", "dummy-api-key")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, mockResponse)
	})

	mockServer := httptest.NewServer(handler)
	return mockServer
}

func TestGetByRegistration(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	BaseURL = mockServer.URL
	client := createTestClient(mockServer)

	registration := "ML58FOU"
	data, err := client.GetByRegistration(registration)
	if err != nil {
		t.Fatalf("GetByRegistration failed: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["registration"] != registration {
		t.Errorf("Expected registration %s, got %s", registration, response["registration"])
	}
}

func TestGetByVIN(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	BaseURL = mockServer.URL
	client := createTestClient(mockServer)

	vin := "BNR32305366"
	data, err := client.GetByVIN(vin)
	if err != nil {
		t.Fatalf("GetByVIN failed: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["vin"] != vin {
		t.Errorf("Expected VIN %s, got %s", vin, response["vin"])
	}
}

func TestGetBulkDownload(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	BaseURL = mockServer.URL
	client := createTestClient(mockServer)

	data, err := client.GetBulkDownload()
	if err != nil {
		t.Fatalf("GetBulkDownload failed: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := response["bulk"]; !ok {
		t.Error("Expected 'bulk' key in response")
	}
	if _, ok := response["delta"]; !ok {
		t.Error("Expected 'delta' key in response")
	}
}

func TestRenewCredentials(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	BaseURL = mockServer.URL
	client := createTestClient(mockServer)

	email := "f@finbarrs.eu"
	apiKey := "dummy-api-key"
	data, err := client.RenewCredentials(apiKey, email)
	if err != nil {
		t.Fatalf("RenewCredentials failed: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal(data, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := response["clientSecret"]; !ok {
		t.Error("Expected 'clientSecret' key in response")
	}
}

func TestInvalidCases(t *testing.T) {
	mockServer := createMockServer()
	defer mockServer.Close()

	BaseURL = mockServer.URL
	client := createTestClient(mockServer)

	t.Run("InvalidRegistration", func(t *testing.T) {
		_, err := client.GetByRegistration("INVALID")
		if err == nil {
			t.Fatal("Expected an error for invalid registration, but got none")
		}
	})

	t.Run("InvalidVIN", func(t *testing.T) {
		_, err := client.GetByVIN("INVALID")
		if err == nil {
			t.Fatal("Expected an error for invalid VIN, but got none")
		}
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		invalidClient := NewClient(ClientConfig{
			ClientID:     "invalid",
			ClientSecret: "invalid",
			APIKey:       "invalid",
		}, nil) // Use `nil` to use the MOT API endpoint

		_, err := invalidClient.GetByRegistration("ML58FOU")
		if err == nil {
			t.Fatal("Expected an error for invalid credentials, but got none")
		}
	})

	t.Run("MaxingBurstLimit", func(t *testing.T) {
		// Need a new client to refresh limiter
		newClient := NewClient(mockConfig, mockServer.Client())

		registration := "ML58FOU"

		for i := 0; i < BurstLimit / 2; i++ { // Must divide burst limit because checking .Allow() costs a token
			allowed := newClient.rateLimiter.Allow()

			if !allowed {
				t.Fatalf("RateLimiter incorrectly blocked request %d during burst", i)
			}
			
			_, err := newClient.GetByRegistration(registration)
			if err != nil {
				t.Fatalf("GetByRegistration failed during burst testing: %v", err)
			}
		}

		time.Sleep(1 * time.Second)

		if !newClient.rateLimiter.Allow() {
			t.Fatalf("RateLimiter token has not replenished.")
		}
	})
}

func createTestClient(mockServer *httptest.Server) *Client {
	mockConfig := ClientConfig{
		ClientID:     "nil",
		ClientSecret: "nil",
		APIKey:       "nil",
	}
	return NewClient(mockConfig, mockServer.Client())
}
