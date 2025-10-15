package controllers

import (
	"delayedNotifier/internal/application/services"
	"delayedNotifier/internal/web_api/models/requests"
	"encoding/json"
	"html/template"
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
	mux.HandleFunc("/", c.View)
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

func (c *NotificationController) View(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("internal/web_api/public/index.html")
	if err != nil {
		http.Error(w, "template not found: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.Execute(w, nil); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Create godoc
// @Summary      Создать уведомление
// @Description  Принимает задачу для отложенной отправки уведомления.
// @Tags         notifications
// @Accept       json
// @Produce      plain
// @Param        request  body      requests.CreateNotificationRequest  true  "Данные уведомления"
// @Success      201      {string}  string  "UUID уведомления"
// @Header       201      {string}  Location  "URL созданного ресурса"
// @Failure      400      {string}  string  "Некорректный запрос"
// @Failure      500      {string}  string  "Ошибка сервера"
// @Router       /notify [post]
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

// Status godoc
// @Summary      Получить статус уведомления
// @Tags         notifications
// @Produce      plain
// @Param        id   path      string  true  "UUID уведомления"
// @Success      200  {string}  string  "Текущий статус"
// @Failure      400  {string}  string  "Некорректный идентификатор"
// @Failure      500  {string}  string  "Ошибка сервера"
// @Router       /notify/{id} [get]
func (c *NotificationController) Status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "notification_id not provided", http.StatusBadRequest)
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

// Cancel godoc
// @Summary      Отменить уведомление
// @Tags         notifications
// @Produce      plain
// @Param        id   path      string  true  "UUID уведомления"
// @Success      200
// @Failure      400  {string}  string  "Некорректный идентификатор"
// @Failure      500  {string}  string  "Ошибка сервера"
// @Router       /notify/{id} [delete]
func (c *NotificationController) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "notification_id not provided", http.StatusBadRequest)
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
