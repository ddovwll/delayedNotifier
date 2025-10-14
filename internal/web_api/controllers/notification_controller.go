package controllers

import (
	"delayedNotifier/internal/application/services"
	"delayedNotifier/internal/web_api/models/requests"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

type NotificationController struct {
	notificationService *services.NotificationService
}

func NewNotificationController(notificationService *services.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
	}
}

func (c *NotificationController) MapRoutes(mux *http.ServeMux) *http.ServeMux {
	mux.HandleFunc("/notify", c.Create)
	mux.HandleFunc("/notify/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == "GET" {
			c.Status(writer, request)
		}
		if request.Method == "DELETE" {
			c.Cancel(writer, request)
		}
	})
	return mux
}

func (c *NotificationController) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	notification := requests.CreateNotificationRequest{}
	err := json.NewDecoder(r.Body).Decode(&notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println(err)
		return
	}

	notificationModel := notification.MapToModel()
	err = c.notificationService.Create(ctx, &notificationModel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Location", "http://"+os.Getenv("HTTP_HOST")+":"+os.Getenv("HTTP_PORT")+"/notify/"+notificationModel.ID.String())
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(notificationModel.ID.String()))
	if err != nil {
		log.Println(err)
	}
}

func (c *NotificationController) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "order_uid not provided", http.StatusBadRequest)
		return
	}

	notificationId, err := uuid.Parse(parts[2])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := c.notificationService.GetStatus(ctx, notificationId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(status))
	if err != nil {
		log.Println(err)
	}
}

func (c *NotificationController) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "order_uid not provided", http.StatusBadRequest)
		return
	}

	notificationId, err := uuid.Parse(parts[2])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = c.notificationService.CancelNotification(ctx, notificationId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
