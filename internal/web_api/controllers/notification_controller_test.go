package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"delayedNotifier/internal/application/contracts"
	"delayedNotifier/internal/application/services"
	"delayedNotifier/internal/domain/models"
	"delayedNotifier/internal/web_api/models/requests"

	"github.com/google/uuid"
)

type testNotificationRepository struct {
	createFunc  func(context.Context, *models.Notification) error
	getByIDFunc func(context.Context, uuid.UUID) (*models.Notification, error)
	updateFunc  func(context.Context, *models.Notification) error

	createCalls  int
	getByIDCalls int
	updateCalls  int
}

func (r *testNotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	r.createCalls++
	if r.createFunc != nil {
		return r.createFunc(ctx, n)
	}
	return nil
}

func (r *testNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	r.getByIDCalls++
	if r.getByIDFunc != nil {
		return r.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (r *testNotificationRepository) Update(ctx context.Context, n *models.Notification) error {
	r.updateCalls++
	if r.updateFunc != nil {
		return r.updateFunc(ctx, n)
	}
	return nil
}

func (r *testNotificationRepository) Delete(context.Context, uuid.UUID) error { return nil }

type testProducer struct {
	publishFunc func(string, []byte, time.Duration) error
	calls       int
}

func (p *testProducer) Publish(routingKey string, message []byte, delay time.Duration) error {
	p.calls++
	if p.publishFunc != nil {
		return p.publishFunc(routingKey, message, delay)
	}
	return nil
}

type testNotifierFactory struct {
	getFunc func(models.Channel) (contracts.Notifier, error)
}

func (f *testNotifierFactory) GetNotifier(channel models.Channel) (contracts.Notifier, error) {
	if f.getFunc != nil {
		return f.getFunc(channel)
	}
	return nil, errors.New("no notifier")
}

type testRetryer struct {
	retryFunc func(func() error) error
	calls     int
}

func (r *testRetryer) Retry(fn func() error) error {
	r.calls++
	if r.retryFunc != nil {
		return r.retryFunc(fn)
	}
	return fn()
}

type testCache struct {
	setFunc func(context.Context, string, interface{}, time.Duration) error
	getFunc func(context.Context, string) (string, error)
}

func (c *testCache) Set(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	if c.setFunc != nil {
		return c.setFunc(ctx, key, value, expiry)
	}
	return nil
}

func (c *testCache) Get(ctx context.Context, key string) (string, error) {
	if c.getFunc != nil {
		return c.getFunc(ctx, key)
	}
	return "", errors.New("miss")
}

type controllerEnv struct {
	repo       *testNotificationRepository
	producer   *testProducer
	factory    *testNotifierFactory
	retryer    *testRetryer
	cache      *testCache
	service    *services.NotificationService
	controller *NotificationController
}

func newControllerEnv(setup func(env *controllerEnv)) *controllerEnv {
	env := &controllerEnv{
		repo:     &testNotificationRepository{},
		producer: &testProducer{},
		factory:  &testNotifierFactory{},
		retryer:  &testRetryer{},
		cache:    &testCache{},
	}

	if setup != nil {
		setup(env)
	}

	deliverySvc := services.NewDeliveryTaskService(env.producer, "notifications")

	env.service = services.NewNotificationService(env.repo, deliverySvc, env.factory, env.retryer, env.cache)
	env.controller = NewNotificationController(env.service)

	return env
}

func TestNotificationController_CreateSuccess(t *testing.T) {
	env := newControllerEnv(nil)
	reqPayload := requests.CreateNotificationRequest{
		Channel:     models.Email,
		Recipient:   "user@example.com",
		Message:     "Hello",
		ScheduledAt: time.Now().Add(time.Hour),
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	t.Setenv("HTTP_HOST", "localhost")
	t.Setenv("HTTP_PORT", "8080")

	env.controller.Create(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	if env.repo.createCalls != 1 {
		t.Fatalf("expected repository Create to be called once, got %d", env.repo.createCalls)
	}

	if env.producer.calls != 1 {
		t.Fatalf("expected publisher to be called once, got %d", env.producer.calls)
	}

	if rec.Body.Len() == 0 {
		t.Fatalf("expected response body with notification ID")
	}

	location := resp.Header.Get("Location")
	if location == "" {
		t.Fatalf("expected Location header to be set")
	}
}

func TestNotificationController_CreateInvalidJSON(t *testing.T) {
	env := newControllerEnv(nil)

	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString("invalid"))
	rec := httptest.NewRecorder()

	env.controller.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	if env.repo.createCalls != 0 {
		t.Fatalf("Create should not be called on invalid JSON")
	}
}

func TestNotificationController_CreateValidationError(t *testing.T) {
	env := newControllerEnv(nil)
	reqPayload := requests.CreateNotificationRequest{
		Channel:     models.Email,
		Recipient:   "",
		Message:     "",
		ScheduledAt: time.Now().Add(-time.Minute),
	}

	body, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	env.controller.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	if env.repo.createCalls != 0 {
		t.Fatalf("Create should not be called when validation fails")
	}

	if rec.Body.Len() == 0 {
		t.Fatalf("expected validation error message in response body")
	}
}

func TestNotificationController_CreateServiceError(t *testing.T) {
	expectedErr := errors.New("create failed")
	env := newControllerEnv(func(env *controllerEnv) {
		env.repo.createFunc = func(context.Context, *models.Notification) error {
			return expectedErr
		}
	})

	reqPayload := requests.CreateNotificationRequest{
		Channel:     models.Email,
		Recipient:   "user@example.com",
		Message:     "Hello",
		ScheduledAt: time.Now().Add(time.Hour),
	}

	body, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	env.controller.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestNotificationController_StatusSuccess(t *testing.T) {
	notification := &models.Notification{
		ID:     uuid.New(),
		Status: models.Sent,
	}

	env := newControllerEnv(func(env *controllerEnv) {
		env.cache.getFunc = func(context.Context, string) (string, error) {
			bytes, _ := json.Marshal(notification)
			return string(bytes), nil
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/notify/"+notification.ID.String(), nil)
	rec := httptest.NewRecorder()

	env.controller.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if rec.Body.String() != notification.Status.String() {
		t.Fatalf("expected body %q, got %q", notification.Status.String(), rec.Body.String())
	}
}

func TestNotificationController_StatusInvalidUUID(t *testing.T) {
	env := newControllerEnv(nil)
	req := httptest.NewRequest(http.MethodGet, "/notify/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	env.controller.Status(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_StatusServiceError(t *testing.T) {
	env := newControllerEnv(func(env *controllerEnv) {
		env.cache.getFunc = func(context.Context, string) (string, error) {
			return "", errors.New("miss")
		}
		env.repo.getByIDFunc = func(context.Context, uuid.UUID) (*models.Notification, error) {
			return nil, errors.New("db failure")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/notify/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	env.controller.Status(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestNotificationController_CancelSuccess(t *testing.T) {
	notification := &models.Notification{ID: uuid.New()}
	env := newControllerEnv(func(env *controllerEnv) {
		env.repo.getByIDFunc = func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		}
		env.repo.updateFunc = func(ctx context.Context, n *models.Notification) error {
			if n.Status != models.Cancelled {
				t.Fatalf("expected Cancelled status, got %v", n.Status)
			}
			return nil
		}
		env.cache.getFunc = func(context.Context, string) (string, error) {
			return "cached", nil
		}
	})

	req := httptest.NewRequest(http.MethodDelete, "/notify/"+notification.ID.String(), nil)
	rec := httptest.NewRecorder()

	env.controller.Cancel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if env.repo.updateCalls != 1 {
		t.Fatalf("expected Update to be called once, got %d", env.repo.updateCalls)
	}
}

func TestNotificationController_CancelInvalidUUID(t *testing.T) {
	env := newControllerEnv(nil)
	req := httptest.NewRequest(http.MethodDelete, "/notify/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	env.controller.Cancel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestNotificationController_CancelServiceError(t *testing.T) {
	env := newControllerEnv(func(env *controllerEnv) {
		env.repo.getByIDFunc = func(context.Context, uuid.UUID) (*models.Notification, error) {
			return nil, errors.New("db failure")
		}
	})

	req := httptest.NewRequest(http.MethodDelete, "/notify/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()

	env.controller.Cancel(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
