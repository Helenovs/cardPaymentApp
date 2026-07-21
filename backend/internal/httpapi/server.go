package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"card-payment-app/backend/internal/payments"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/paymentintent"
	"github.com/stripe/stripe-go/v85/webhook"
)

const (
	demoAmount   int64 = 2500
	demoCurrency       = "aud"
	maxBodyBytes int64 = 64 * 1024
)

type Config struct {
	PublishableKey string
	WebhookSecret  string
	AllowedOrigin  string
}

type Server struct {
	store        payments.Store
	config       Config
	logger       *slog.Logger
	createIntent func(*stripe.PaymentIntentParams) (*stripe.PaymentIntent, error)
}

func New(store payments.Store, config Config, logger *slog.Logger) *Server {
	return &Server{store: store, config: config, logger: logger, createIntent: paymentintent.New}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/config", s.getConfig)
	mux.HandleFunc("POST /api/payment-intents", s.createPaymentIntent)
	mux.HandleFunc("GET /api/payments", s.listPayments)
	mux.HandleFunc("POST /api/webhooks/stripe", s.stripeWebhook)
	return s.cors(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"publishableKey": s.config.PublishableKey, "amount": demoAmount, "currency": demoCurrency})
}

func (s *Server) createPaymentIntent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Email = strings.TrimSpace(body.Email)
	if body.Email == "" || !strings.Contains(body.Email, "@") {
		writeError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	params := &stripe.PaymentIntentParams{Amount: stripe.Int64(demoAmount), Currency: stripe.String(string(stripe.CurrencyAUD)), ReceiptEmail: stripe.String(body.Email)}
	params.AutomaticPaymentMethods = &stripe.PaymentIntentAutomaticPaymentMethodsParams{Enabled: stripe.Bool(true)}
	params.AddMetadata("source", "card-payment-app")
	intent, err := s.createIntent(params)
	if err != nil {
		s.logger.Error("create Stripe PaymentIntent", "error", err)
		writeError(w, http.StatusBadGateway, "could not start payment")
		return
	}
	now := time.Now().UTC()
	payment := paymentFromIntent(intent, now)
	payment.CustomerEmail = body.Email
	if err := s.store.Save(r.Context(), payment); err != nil {
		s.logger.Error("store PaymentIntent", "error", err)
		writeError(w, http.StatusServiceUnavailable, "could not store payment")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"clientSecret": intent.ClientSecret, "paymentIntentId": intent.ID})
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	items, err := s.store.List(r.Context(), limit)
	if err != nil {
		s.logger.Error("list payments", "error", err)
		writeError(w, http.StatusServiceUnavailable, "could not load payments")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": items})
}

func (s *Server) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	if s.config.WebhookSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "webhook secret is not configured")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook body")
		return
	}
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), s.config.WebhookSecret)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook signature")
		return
	}
	if !strings.HasPrefix(string(event.Type), "payment_intent.") {
		_, _ = s.store.MarkEventProcessed(r.Context(), event.ID)
		writeJSON(w, http.StatusOK, map[string]bool{"received": true})
		return
	}
	var intent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &intent); err != nil {
		writeError(w, http.StatusBadRequest, "invalid PaymentIntent event")
		return
	}
	payment := paymentFromIntent(&intent, time.Now().UTC())
	payment.LastEventID = event.ID
	payment.LastEventType = string(event.Type)
	if err := s.store.Save(r.Context(), payment); err != nil {
		s.logger.Error("store webhook payment", "error", err)
		writeError(w, http.StatusServiceUnavailable, "could not store payment")
		return
	}
	// Mark the event only after its payment state is durable. Duplicate Stripe
	// deliveries remain safe because saving a PaymentIntent is an idempotent upsert.
	if _, err := s.store.MarkEventProcessed(r.Context(), event.ID); err != nil {
		s.logger.Error("mark webhook processed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "could not finish webhook")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"received": true})
}

func paymentFromIntent(intent *stripe.PaymentIntent, updated time.Time) payments.Payment {
	created := time.Unix(intent.Created, 0).UTC()
	if intent.Created == 0 {
		created = updated
	}
	p := payments.Payment{ID: intent.ID, Amount: intent.Amount, Currency: string(intent.Currency), Status: string(intent.Status), CreatedAt: created, UpdatedAt: updated, CustomerEmail: intent.ReceiptEmail}
	if intent.PaymentMethod != nil {
		p.PaymentMethod = intent.PaymentMethod.ID
	}
	if intent.LastPaymentError != nil {
		p.LastError = intent.LastPaymentError.Msg
	}
	return p
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin != s.config.AllowedOrigin {
			writeError(w, http.StatusForbidden, "origin is not allowed")
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Stripe-Signature")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
