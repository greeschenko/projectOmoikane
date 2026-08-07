package audit

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Event struct {
	UserID     uint   `json:"userId"`
	UserName   string `json:"userName"`
	Action     string `json:"action"`
	EntityType string `json:"entityType"`
	EntityID   uint   `json:"entityId,omitempty"`
	Detail     string `json:"detail,omitempty"`
	IP         string `json:"ip,omitempty"`
	UserAgent  string `json:"userAgent,omitempty"`
}

var Emit = func(serviceURL string, event Event) {
	if serviceURL == "" {
		return
	}
	go func() {
		body, err := json.Marshal(event)
		if err != nil {
			log.Printf("[audit] marshal error: %v", err)
			return
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Post(serviceURL+"/events", "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("[audit] emit failed: %v", err)
			return
		}
		resp.Body.Close()
	}()
}
