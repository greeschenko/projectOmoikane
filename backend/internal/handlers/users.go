package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"omoikane-backend/internal/audit"
	"omoikane-backend/internal/middleware"
	"omoikane-backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type updateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
	Status   *string `json:"status,omitempty"`
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var users []models.User
	if err := h.DB.Find(&users).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch users"})
		return
	}

	result := make([]map[string]interface{}, len(users))
	for i, u := range users {
		result[i] = sanitizeUserJSON(u)
	}

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "All fields required"})
		return
	}

	if errMsg := validatePassword(req.Password); errMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		return
	}

	var count int64
	h.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email already registered"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to hash password"})
		return
	}

	role := req.Role
	if role == "" {
		role = "user"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}

	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashed),
		Role:     role,
		Status:   status,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
		return
	}

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "create",
		EntityType: "user",
		EntityID:   user.ID,
		Detail:     fmt.Sprintf("Created user %s (%s)", user.Name, user.Email),
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sanitizeUserJSON(user))
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "User ID required"})
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, idStr).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Password != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to hash password"})
			return
		}
		updates["password"] = string(hashed)
	}

	if err := h.DB.Model(&user).Updates(updates).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update user"})
		return
	}

	h.DB.First(&user, user.ID)

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "update",
		EntityType: "user",
		EntityID:   user.ID,
		Detail:     fmt.Sprintf("Updated user %s", user.Name),
	})

	json.NewEncoder(w).Encode(sanitizeUserJSON(user))
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "User ID required"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, idStr).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

	if err := h.DB.Delete(&user).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete user"})
		return
	}

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "delete",
		EntityType: "user",
		EntityID:   user.ID,
		Detail:     fmt.Sprintf("Deleted user %s", user.Name),
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) BatchUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Action string `json:"action"`
		IDs    []uint `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if len(req.IDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No IDs provided"})
		return
	}

	switch req.Action {
	case "delete":
		h.DB.Delete(&models.User{}, req.IDs)
	case "ban":
		h.DB.Model(&models.User{}).Where("id IN ?", req.IDs).Update("status", "banned")
	case "activate":
		h.DB.Model(&models.User{}).Where("id IN ?", req.IDs).Update("status", "active")
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unknown action"})
		return
	}

	actorID := middleware.GetUserID(r)
	var actorName string
	if actorID > 0 {
		var actor models.User
		h.DB.First(&actor, actorID)
		actorName = actor.Name
	} else {
		actorName = "system"
	}
	audit.Emit(h.AuditServiceURL, audit.Event{
		UserID:     actorID,
		UserName:   actorName,
		Action:     "batch_" + req.Action,
		EntityType: "user",
		Detail:     fmt.Sprintf("Batch %s on %d users", req.Action, len(req.IDs)),
	})

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
