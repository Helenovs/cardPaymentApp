package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"card-payment-app/backend/internal/payments"
	"github.com/stripe/stripe-go/v85"
)

type fakeStore struct {
	items []payments.Payment
	saved *payments.Payment
}

func (f *fakeStore) Save(_ context.Context, payment payments.Payment) error {
	f.saved = &payment
	return nil
}
func (f *fakeStore) List(_ context.Context, _ int64) ([]payments.Payment, error)  { return f.items, nil }
func (f *fakeStore) MarkEventProcessed(_ context.Context, _ string) (bool, error) { return true, nil }

func testServer(store payments.Store) *Server {
	return New(store, Config{PublishableKey: "pk_test_example", AllowedOrigin: "http://localhost:5173"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestGetConfig(t *testing.T) {
	recorder := httptest.NewRecorder()
	testServer(&fakeStore{}).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/config", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["publishableKey"] != "pk_test_example" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["amount"] != float64(2500) {
		t.Fatalf("amount = %v", response["amount"])
	}
}

func TestCreatePaymentIntent(t *testing.T) {
	store := &fakeStore{}
	server := testServer(store)
	server.createIntent = func(params *stripe.PaymentIntentParams) (*stripe.PaymentIntent, error) {
		if *params.Amount != 2500 {
			t.Fatalf("amount = %d", *params.Amount)
		}
		return &stripe.PaymentIntent{ID: "pi_test", ClientSecret: "pi_test_secret", Amount: 2500, Currency: stripe.CurrencyAUD, Status: stripe.PaymentIntentStatusRequiresPaymentMethod, Created: time.Now().Unix()}, nil
	}
	request := httptest.NewRequest(http.MethodPost, "/api/payment-intents", strings.NewReader(`{"email":"buyer@example.com"}`))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if store.saved == nil || store.saved.ID != "pi_test" || store.saved.CustomerEmail != "buyer@example.com" {
		t.Fatalf("saved = %#v", store.saved)
	}
}

func TestRejectsUnexpectedOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/payments", nil)
	request.Header.Set("Origin", "https://attacker.example")
	recorder := httptest.NewRecorder()
	testServer(&fakeStore{}).Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d", recorder.Code)
	}
}
