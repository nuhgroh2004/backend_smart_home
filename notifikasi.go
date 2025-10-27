package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type Notifikasi struct {
	AsapTerdeteksi bool   `json:"asapTerdeteksi"`
	Pesan          string `json:"pesan"`
	TandonPenuh    bool   `json:"tandonPenuh"`
}
type FCMToken struct {
	Token       string `json:"token"`
	DeviceName  string `json:"deviceName"`
	Active      bool   `json:"active"`
	LastUpdated string `json:"lastUpdated"`
}

type RegisterRequest struct {
	DeviceID   string `json:"device_id"`
	FCMToken   string `json:"fcm_token"`
	DeviceName string `json:"device_name"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	dbClient        *db.Client
	messagingClient *messaging.Client
	previousState   Notifikasi
	apiDBClient     *db.Client
)

const (
	credentialPath = "serviceAccountKey.json"
	databaseURL    = "https://smarthome-a3fc2-default-rtdb.firebaseio.com"
	checkInterval  = 2 * time.Second
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "api" {
		startAPIServer()
	} else {
		runMonitoringService()
	}
}

func typePrintln(s string, charDelay time.Duration) {
	for _, r := range s {
		fmt.Printf("%c", r)
		time.Sleep(charDelay)
	}
	fmt.Println()
}

func runMonitoringService() {
	ctx := context.Background()
	if err := initFirebase(ctx); err != nil {
		log.Fatalf("❌ Error initializing Firebase: %v", err)
	}

	lines := []string{
		"",
		"<====================== Golang 1.24 ======================>",
		"",
		"Fokuslah pada pengguna, dan semua hal lain akan mengikuti.",
		"",
		"server menyala ............................................",
		"",
	}

	for _, line := range lines {
		if line == "" {
			fmt.Println()
		} else {
			typePrintln(line, 40*time.Millisecond)
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := loadInitialState(ctx); err != nil {
		log.Fatalf("❌ Error loading initial state: %v", err)
	}
	monitorNotifications(ctx)
}

func initFirebase(ctx context.Context) error {
	opt := option.WithCredentialsFile(credentialPath)
	config := &firebase.Config{
		DatabaseURL: databaseURL,
	}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return fmt.Errorf("error initializing app: %v", err)
	}
	client, err := app.Database(ctx)
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	dbClient = client
	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("error initializing messaging: %v", err)
	}
	messagingClient = msgClient
	return nil
}

func loadInitialState(ctx context.Context) error {
	ref := dbClient.NewRef("IoTSystem/Notifikasi")
	if err := ref.Get(ctx, &previousState); err != nil {
		return err
	}
	return nil
}

func getCurrentState(ctx context.Context) (Notifikasi, error) {
	var current Notifikasi
	ref := dbClient.NewRef("IoTSystem/Notifikasi")
	if err := ref.Get(ctx, &current); err != nil {
		return current, err
	}
	return current, nil
}

func getActiveFCMTokens(ctx context.Context) ([]string, error) {
	var tokensMap map[string]FCMToken
	ref := dbClient.NewRef("IoTSystem/FCMTokens")
	if err := ref.Get(ctx, &tokensMap); err != nil {
		return nil, err
	}
	var activeTokens []string
	for _, tokenData := range tokensMap {
		if tokenData.Active && tokenData.Token != "" {
			activeTokens = append(activeTokens, tokenData.Token)
		}
	}
	return activeTokens, nil
}

func sendNotification(ctx context.Context, title, body, notifType string) error {
	tokens, err := getActiveFCMTokens(ctx)
	if err != nil {
		return fmt.Errorf("error getting tokens: %v", err)
	}
	if len(tokens) == 0 {
		return fmt.Errorf("no active FCM tokens found")
	}
	successCount := 0
	for _, token := range tokens {
		message := &messaging.Message{
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: map[string]string{
				"type":      notifType,
				"timestamp": time.Now().Format(time.RFC3339),
			},
			Token: token,
		}
		if notifType == "asap_terdeteksi" {
			message.Android = &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					Sound:     "alarm",
					ChannelID: "smarthome_alerts",
					Priority:  messaging.PriorityHigh,
				},
			}
		}
		_, err := messagingClient.Send(ctx, message)
		if err != nil {
			continue
		}
		successCount++
	}
	if successCount > 0 {
		return nil
	}
	return fmt.Errorf("failed to send to any device")
}

func checkAndNotify(ctx context.Context, current Notifikasi) {
	if current.AsapTerdeteksi != previousState.AsapTerdeteksi && current.AsapTerdeteksi {
		err := sendNotification(ctx,
			"Sensor Asap",
			"Asap terdeteksi berpotensi kebakaran",
			"asap_terdeteksi",
		)
		if err == nil {
			fmt.Println("Notifikasi sensor asap terkirim (✓)")
		}
	}
	if current.TandonPenuh != previousState.TandonPenuh && current.TandonPenuh {
		err := sendNotification(ctx,
			"Tandon Air",
			"Tandon Air penuh",
			"tandon_penuh",
		)
		if err == nil {
			fmt.Println("Notifikasi tandon air terkirim (✓)")
		}
	}
}

func monitorNotifications(ctx context.Context) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for range ticker.C {
		currentState, err := getCurrentState(ctx)
		if err != nil {
			continue
		}
		checkAndNotify(ctx, currentState)
		previousState = currentState
	}
}

// ============================================
// API SERVER FUNCTIONS
// ============================================

func initAPIFirebase() error {
	ctx := context.Background()
	opt := option.WithCredentialsFile(credentialPath)
	config := &firebase.Config{
		DatabaseURL: databaseURL,
	}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return err
	}
	client, err := app.Database(ctx)
	if err != nil {
		return err
	}
	apiDBClient = client
	return nil
}

func registerFCMHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}
	if req.DeviceID == "" || req.FCMToken == "" {
		sendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "device_id and fcm_token are required",
		})
		return
	}
	ctx := context.Background()
	ref := apiDBClient.NewRef("IoTSystem/FCMTokens/" + req.DeviceID)
	tokenData := map[string]interface{}{
		"token":       req.FCMToken,
		"deviceName":  req.DeviceName,
		"lastUpdated": time.Now().Format(time.RFC3339),
		"active":      true,
		"createdAt":   time.Now().Format(time.RFC3339),
	}
	if err := ref.Set(ctx, tokenData); err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to register token",
		})
		return
	}
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "FCM token registered successfully",
		Data: map[string]string{
			"device_id":     req.DeviceID,
			"device_name":   req.DeviceName,
			"registered_at": time.Now().Format(time.RFC3339),
		},
	})
}

func getNotificationStatusHandler(w http.ResponseWriter, _ *http.Request) {
	ctx := context.Background()
	ref := apiDBClient.NewRef("IoTSystem/Notifikasi")
	var notif Notifikasi
	if err := ref.Get(ctx, &notif); err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to fetch notification status",
		})
		return
	}
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    notif,
	})
}

func getTokensHandler(w http.ResponseWriter, _ *http.Request) {
	ctx := context.Background()
	ref := apiDBClient.NewRef("IoTSystem/FCMTokens")
	var tokens map[string]FCMToken
	if err := ref.Get(ctx, &tokens); err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "Failed to fetch tokens",
		})
		return
	}
	sendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    tokens,
	})
}

func sendJSONResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func startAPIServer() {
	if err := initAPIFirebase(); err != nil {
		log.Fatalf("Failed to initialize Firebase for API: %v", err)
	}
	http.HandleFunc("/api/fcm/register", enableCORS(registerFCMHandler))
	http.HandleFunc("/api/fcm/notification-status", enableCORS(getNotificationStatusHandler))
	http.HandleFunc("/api/fcm/tokens", enableCORS(getTokensHandler))
	fmt.Println("🌐 API Server started on http://localhost:8080")
	fmt.Println("   POST /api/fcm/register")
	fmt.Println("   GET  /api/fcm/notification-status")
	fmt.Println("   GET  /api/fcm/tokens")
	fmt.Println()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

//Dependencies yang Perlu Diinstall
//Dependencies yang Perlu Diinstall:
//go get firebase.google.com/go/v4
//go get firebase.google.com/go/v4/messaging
//go get google.golang.org/api/option
//go get cloud.google.com/go/firestore
