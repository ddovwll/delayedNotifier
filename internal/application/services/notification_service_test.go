package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"delayedNotifier/internal/application/contracts"
	"delayedNotifier/internal/domain/models"

	"github.com/google/uuid"
)

type stubNotificationRepository struct {
	createFunc       func(context.Context, *models.Notification) error
	getByIDFunc      func(context.Context, uuid.UUID) (*models.Notification, error)
	updateFunc       func(context.Context, *models.Notification) error
	deleteFunc       func(context.Context, uuid.UUID) error
	updateStatusFunc func(context.Context, uuid.UUID, models.Status, models.Status, time.Time) (bool, error)

	createCalls       int
	getByIDCalls      int
	updateCalls       int
	updateStatusCalls int
}

func (s *stubNotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	s.createCalls++
	if s.createFunc != nil {
		return s.createFunc(ctx, n)
	}
	return nil
}

func (s *stubNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	s.getByIDCalls++
	if s.getByIDFunc != nil {
		return s.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (s *stubNotificationRepository) Update(ctx context.Context, n *models.Notification) error {
	s.updateCalls++
	if s.updateFunc != nil {
		return s.updateFunc(ctx, n)
	}
	return nil
}

func (s *stubNotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(ctx, id)
	}
	return nil
}

func (s *stubNotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, current, next models.Status, updatedAt time.Time) (bool, error) {
	s.updateStatusCalls++
	if s.updateStatusFunc != nil {
		return s.updateStatusFunc(ctx, id, current, next, updatedAt)
	}
	return true, nil
}

type stubCache struct {
	getFunc func(context.Context, string) (string, error)
	setFunc func(context.Context, string, interface{}, time.Duration) error

	getCalls []string
	setCalls []cacheSetArgs
}

type cacheSetArgs struct {
	key    string
	value  interface{}
	expiry time.Duration
}

func (s *stubCache) Get(ctx context.Context, key string) (string, error) {
	s.getCalls = append(s.getCalls, key)
	if s.getFunc != nil {
		return s.getFunc(ctx, key)
	}
	return "", errors.New("not implemented")
}

func (s *stubCache) Set(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	s.setCalls = append(s.setCalls, cacheSetArgs{key: key, value: value, expiry: expiry})
	if s.setFunc != nil {
		return s.setFunc(ctx, key, value, expiry)
	}
	return nil
}

type stubRetryer struct {
	retryFunc func(func() error) error
	calls     int
}

func (s *stubRetryer) Retry(fn func() error) error {
	s.calls++
	if s.retryFunc != nil {
		return s.retryFunc(fn)
	}
	return fn()
}

type stubNotifierFactory struct {
	getFunc func(models.Channel) (contracts.Notifier, error)
}

func (s *stubNotifierFactory) GetNotifier(channel models.Channel) (contracts.Notifier, error) {
	if s.getFunc != nil {
		return s.getFunc(channel)
	}
	return nil, errors.New("not implemented")
}

type stubNotifier struct {
	notifyFunc func(context.Context, string, string) error
	calls      int
	lastCtx    context.Context
	lastTo     string
	lastMsg    string
}

func (s *stubNotifier) Notify(ctx context.Context, recipient, message string) error {
	s.calls++
	s.lastCtx = ctx
	s.lastTo = recipient
	s.lastMsg = message
	if s.notifyFunc != nil {
		return s.notifyFunc(ctx, recipient, message)
	}
	return nil
}

type stubProducer struct {
	publishFunc func(string, []byte, time.Duration) error
	calls       int
	lastKey     string
	lastMsg     []byte
	lastDelay   time.Duration
}

func (s *stubProducer) Publish(routingKey string, message []byte, delay time.Duration) error {
	s.calls++
	s.lastKey = routingKey
	s.lastMsg = message
	s.lastDelay = delay
	if s.publishFunc != nil {
		return s.publishFunc(routingKey, message, delay)
	}
	return nil
}

func newNotification(id uuid.UUID, status models.Status) *models.Notification {
	return &models.Notification{
		ID:          id,
		Channel:     models.Email,
		Recipient:   "user@example.org",
		Message:     "hello",
		ScheduledAt: time.Now().Add(5 * time.Minute),
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestNotificationService_CreateSuccess(t *testing.T) {
	ctx := context.Background()
	repo := &stubNotificationRepository{}

	producer := &stubProducer{}
	deliverySvc := &DeliveryTaskService{producer: producer, routingKey: "delivery"}

	svc := &NotificationService{
		notificationRepository: repo,
		deliveryTaskService:    deliverySvc,
	}

	notification := newNotification(uuid.New(), models.Scheduled)

	if err := svc.Create(ctx, notification); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if repo.createCalls != 1 {
		t.Fatalf("expected repository Create to be called once, got %d", repo.createCalls)
	}

	if producer.calls != 1 {
		t.Fatalf("expected producer Publish to be called once, got %d", producer.calls)
	}

	var published models.DeliveryTask
	if err := json.Unmarshal(producer.lastMsg, &published); err != nil {
		t.Fatalf("failed to unmarshal published message: %v", err)
	}

	if published.NotificationID != notification.ID {
		t.Fatalf("expected task NotificationID %s, got %s", notification.ID, published.NotificationID)
	}

	if producer.lastKey != "delivery" {
		t.Fatalf("expected routing key 'delivery', got %q", producer.lastKey)
	}

	expectedDelay := time.Until(notification.ScheduledAt)
	if diff := expectedDelay - producer.lastDelay; diff < -10*time.Millisecond || diff > 10*time.Millisecond {
		t.Fatalf("unexpected delay: want %v, got %v", expectedDelay, producer.lastDelay)
	}
}

func TestNotificationService_CreateRepositoryError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db error")
	repo := &stubNotificationRepository{
		createFunc: func(context.Context, *models.Notification) error {
			return expectedErr
		},
	}

	producer := &stubProducer{
		publishFunc: func(string, []byte, time.Duration) error {
			t.Fatalf("Publish should not be called when repository.Create fails")
			return nil
		},
	}

	deliverySvc := &DeliveryTaskService{producer: producer, routingKey: "delivery"}

	svc := &NotificationService{
		notificationRepository: repo,
		deliveryTaskService:    deliverySvc,
	}

	err := svc.Create(ctx, newNotification(uuid.New(), models.Scheduled))
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if producer.calls != 0 {
		t.Fatalf("producer should not be called, got %d calls", producer.calls)
	}
}

func TestNotificationService_CreatePublishError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("publish failed")
	repo := &stubNotificationRepository{}

	producer := &stubProducer{
		publishFunc: func(string, []byte, time.Duration) error {
			return expectedErr
		},
	}

	deliverySvc := &DeliveryTaskService{producer: producer, routingKey: "delivery"}

	svc := &NotificationService{
		notificationRepository: repo,
		deliveryTaskService:    deliverySvc,
	}

	err := svc.Create(ctx, newNotification(uuid.New(), models.Scheduled))
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if repo.createCalls != 1 {
		t.Fatalf("expected repository Create to be called once, got %d", repo.createCalls)
	}
}

func TestNotificationService_CreateValidationError(t *testing.T) {
	ctx := context.Background()
	repo := &stubNotificationRepository{}
	producer := &stubProducer{}
	deliverySvc := &DeliveryTaskService{producer: producer, routingKey: "delivery"}

	svc := &NotificationService{
		notificationRepository: repo,
		deliveryTaskService:    deliverySvc,
	}

	notification := newNotification(uuid.New(), models.Scheduled)
	notification.Message = ""

	err := svc.Create(ctx, notification)
	if !errors.Is(err, ErrNotificationMessageRequired) {
		t.Fatalf("expected error %v, got %v", ErrNotificationMessageRequired, err)
	}

	notification.Message = "valid"
	notification.ScheduledAt = time.Now().Add(-time.Minute)
	err = svc.Create(ctx, notification)
	if !errors.Is(err, ErrNotificationScheduledInPast) {
		t.Fatalf("expected error %v, got %v", ErrNotificationScheduledInPast, err)
	}

	if repo.createCalls != 0 {
		t.Fatalf("repository Create should not be called when validation fails")
	}
}

func TestNotificationService_GetStatusCacheHit(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Sent)

	cache := &stubCache{
		getFunc: func(context.Context, string) (string, error) {
			bytes, _ := json.Marshal(notification)
			return string(bytes), nil
		},
	}

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			t.Fatalf("repository GetByID should not be called when cache hits")
			return nil, nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		cache:                  cache,
	}

	status, err := svc.GetStatus(ctx, notification.ID)
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}

	if status != notification.Status.String() {
		t.Fatalf("expected status %q, got %q", notification.Status.String(), status)
	}

	if len(cache.getCalls) != 1 {
		t.Fatalf("expected cache.Get to be called once, got %d", len(cache.getCalls))
	}

	if len(cache.setCalls) != 0 {
		t.Fatalf("cache.Set should not be called on hit")
	}
}

func TestNotificationService_GetStatusCacheMiss(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Failed)

	cache := &stubCache{
		getFunc: func(context.Context, string) (string, error) {
			return "", errors.New("miss")
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
			if expiry != 2*time.Hour {
				t.Fatalf("unexpected cache expiry: %v", expiry)
			}
			strValue, ok := value.(string)
			if !ok {
				t.Fatalf("expected cached value to be string, got %T", value)
			}
			var cached models.Notification
			if err := json.Unmarshal([]byte(strValue), &cached); err != nil {
				t.Fatalf("cache stored invalid json: %v", err)
			}
			if cached.ID != notification.ID {
				t.Fatalf("cached notification mismatches: %s vs %s", cached.ID, notification.ID)
			}
			return nil
		},
	}

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		cache:                  cache,
	}

	status, err := svc.GetStatus(ctx, notification.ID)
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}

	if status != notification.Status.String() {
		t.Fatalf("expected status %q, got %q", notification.Status.String(), status)
	}

	if repo.getByIDCalls != 1 {
		t.Fatalf("expected repository GetByID to be called once, got %d", repo.getByIDCalls)
	}

	if len(cache.setCalls) != 1 {
		t.Fatalf("expected cache.Set to be called once, got %d", len(cache.setCalls))
	}
}

func TestNotificationService_GetStatusRepositoryError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db")

	cache := &stubCache{
		getFunc: func(context.Context, string) (string, error) {
			return "", errors.New("miss")
		},
	}

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return nil, expectedErr
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		cache:                  cache,
	}

	_, err := svc.GetStatus(ctx, uuid.New())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestNotificationService_CancelNotificationSuccess(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Scheduled)

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
		updateStatusFunc: func(ctx context.Context, id uuid.UUID, current, next models.Status, updatedAt time.Time) (bool, error) {
			if id != notification.ID {
				t.Fatalf("unexpected id: %v", id)
			}
			if current != models.Scheduled {
				t.Fatalf("expected current status Scheduled, got %v", current)
			}
			if next != models.Cancelled {
				t.Fatalf("expected next status Cancelled, got %v", next)
			}
			if updatedAt.IsZero() {
				t.Fatalf("updatedAt must be set")
			}
			return true, nil
		},
	}

	cache := &stubCache{
		getFunc: func(context.Context, string) (string, error) {
			return "cached", nil
		},
		setFunc: func(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
			if expiry != 2*time.Hour {
				t.Fatalf("unexpected expiry: %v", expiry)
			}
			return nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		cache:                  cache,
	}

	if err := svc.CancelNotification(ctx, notification.ID); err != nil {
		t.Fatalf("CancelNotification returned error: %v", err)
	}

	if repo.getByIDCalls != 1 || repo.updateStatusCalls != 1 {
		t.Fatalf("expected repo GetByID and UpdateStatus to be called once; got %d and %d", repo.getByIDCalls, repo.updateStatusCalls)
	}

	if len(cache.setCalls) != 1 {
		t.Fatalf("expected cache.Set to be called once, got %d", len(cache.setCalls))
	}
}

func TestNotificationService_CancelNotificationGetError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("missing")

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return nil, expectedErr
		},
		updateStatusFunc: func(context.Context, uuid.UUID, models.Status, models.Status, time.Time) (bool, error) {
			t.Fatalf("UpdateStatus should not be called when GetByID fails")
			return false, nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		cache:                  &stubCache{},
	}

	err := svc.CancelNotification(ctx, uuid.New())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestNotificationService_CancelNotificationAlreadyProcessed(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Sent)
	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		cache:                  &stubCache{},
	}

	err := svc.CancelNotification(ctx, notification.ID)
	if !errors.Is(err, ErrNotificationAlreadyProcessed) {
		t.Fatalf("expected error %v, got %v", ErrNotificationAlreadyProcessed, err)
	}

	if repo.updateStatusCalls != 0 {
		t.Fatalf("repository UpdateStatus should not be called when notification already processed")
	}
}

func TestNotificationService_CancelNotificationAlreadyCancelled(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Cancelled)
	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
		updateStatusFunc: func(context.Context, uuid.UUID, models.Status, models.Status, time.Time) (bool, error) {
			t.Fatalf("UpdateStatus should not be called for already cancelled notification")
			return false, nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		cache:                  &stubCache{},
	}

	if err := svc.CancelNotification(ctx, notification.ID); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestNotificationService_NotifySuccess(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Scheduled)

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
		updateStatusFunc: func(ctx context.Context, id uuid.UUID, current, next models.Status, updatedAt time.Time) (bool, error) {
			if id != notification.ID {
				t.Fatalf("unexpected id: %v", id)
			}
			if current != models.Scheduled {
				t.Fatalf("expected current status Scheduled, got %v", current)
			}
			if next != models.Sent {
				t.Fatalf("expected next status Sent, got %v", next)
			}
			if updatedAt.IsZero() {
				t.Fatalf("updatedAt must be set")
			}
			return true, nil
		},
	}

	notifier := &stubNotifier{}

	factory := &stubNotifierFactory{
		getFunc: func(channel models.Channel) (contracts.Notifier, error) {
			if channel != notification.Channel {
				t.Fatalf("unexpected channel: %v", channel)
			}
			return notifier, nil
		},
	}

	retryer := &stubRetryer{
		retryFunc: func(fn func() error) error {
			return fn()
		},
	}

	cache := &stubCache{
		getFunc: func(context.Context, string) (string, error) {
			return "cached", nil
		},
		setFunc: func(context.Context, string, interface{}, time.Duration) error {
			return nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		notifierFactory:        factory,
		retryer:                retryer,
		cache:                  cache,
	}

	task := models.DeliveryTask{NotificationID: notification.ID}

	if err := svc.Notify(ctx, task); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}

	if notifier.calls != 1 {
		t.Fatalf("expected notifier.Notify to be called once, got %d", notifier.calls)
	}

	if repo.updateStatusCalls != 1 {
		t.Fatalf("expected repository UpdateStatus to be called once, got %d", repo.updateStatusCalls)
	}
}

func TestNotificationService_NotifyRetryFailure(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Scheduled)

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
		updateStatusFunc: func(ctx context.Context, id uuid.UUID, current, next models.Status, updatedAt time.Time) (bool, error) {
			if id != notification.ID {
				t.Fatalf("unexpected id: %v", id)
			}
			if current != models.Scheduled {
				t.Fatalf("expected current status Scheduled, got %v", current)
			}
			if next != models.Failed {
				t.Fatalf("expected next status Failed, got %v", next)
			}
			if updatedAt.IsZero() {
				t.Fatalf("updatedAt must be set")
			}
			return true, nil
		},
	}

	notifier := &stubNotifier{
		notifyFunc: func(context.Context, string, string) error {
			return errors.New("notify failed")
		},
	}

	factory := &stubNotifierFactory{
		getFunc: func(models.Channel) (contracts.Notifier, error) {
			return notifier, nil
		},
	}

	expectedErr := errors.New("retry failed")
	retryer := &stubRetryer{
		retryFunc: func(fn func() error) error {
			_ = fn()
			return expectedErr
		},
	}

	cache := &stubCache{
		getFunc: func(context.Context, string) (string, error) {
			return "cached", nil
		},
		setFunc: func(context.Context, string, interface{}, time.Duration) error {
			return nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		notifierFactory:        factory,
		retryer:                retryer,
		cache:                  cache,
	}

	err := svc.Notify(ctx, models.DeliveryTask{NotificationID: notification.ID})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if repo.updateStatusCalls != 1 {
		t.Fatalf("expected repository UpdateStatus to be called once, got %d", repo.updateStatusCalls)
	}

	if notifier.calls != 1 {
		t.Fatalf("expected notifier to be called once, got %d", notifier.calls)
	}
}

func TestNotificationService_NotifyCancelled(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Cancelled)

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		notifierFactory: &stubNotifierFactory{
			getFunc: func(models.Channel) (contracts.Notifier, error) {
				t.Fatalf("GetNotifier should not be called for cancelled notifications")
				return nil, nil
			},
		},
		retryer: &stubRetryer{
			retryFunc: func(func() error) error {
				t.Fatalf("Retry should not be called for cancelled notifications")
				return nil
			},
		},
		cache: &stubCache{},
	}

	err := svc.Notify(ctx, models.DeliveryTask{NotificationID: notification.ID})
	if err == nil || err.Error() != "notification is cancelled" {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestNotificationService_NotifyFactoryError(t *testing.T) {
	ctx := context.Background()
	notification := newNotification(uuid.New(), models.Scheduled)

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return notification, nil
		},
	}

	expectedErr := errors.New("no notifier")
	svc := &NotificationService{
		notificationRepository: repo,
		notifierFactory: &stubNotifierFactory{
			getFunc: func(models.Channel) (contracts.Notifier, error) {
				return nil, expectedErr
			},
		},
		retryer: &stubRetryer{},
		cache:   &stubCache{},
	}

	err := svc.Notify(ctx, models.DeliveryTask{NotificationID: notification.ID})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestNotificationService_NotifyGetError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("db")

	repo := &stubNotificationRepository{
		getByIDFunc: func(context.Context, uuid.UUID) (*models.Notification, error) {
			return nil, expectedErr
		},
	}

	svc := &NotificationService{
		notificationRepository: repo,
		notifierFactory:        &stubNotifierFactory{},
		retryer:                &stubRetryer{},
		cache:                  &stubCache{},
	}

	err := svc.Notify(ctx, models.DeliveryTask{NotificationID: uuid.New()})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
