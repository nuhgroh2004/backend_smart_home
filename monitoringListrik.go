package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"google.golang.org/api/option"
)

type ListrikData struct {
	DayaSaatIni_W float64 `json:"dayaSaatIni_W" firestore:"dayaSaatIni_W"`
	Timestamp     string  `json:"timestamp" firestore:"timestamp"`
}

type ElectricityHistory struct {
	TotalDaya_W     float64   `firestore:"totalDaya_W"`     // Total akumulasi daya dalam sehari
	JumlahPembacaan int       `firestore:"jumlahPembacaan"` // Berapa kali data dibaca/diupdate
	RataRata_W      float64   `firestore:"rataRata_W"`      // Rata-rata daya
	DayaTerakhir_W  float64   `firestore:"dayaTerakhir_W"`  // Daya terakhir yang tercatat
	Date            string    `firestore:"date"`            // Format: 2025-10-27
	Year            int       `firestore:"year"`
	Month           int       `firestore:"month"`
	Day             int       `firestore:"day"`
	FirstRecordedAt time.Time `firestore:"firstRecordedAt"` // Waktu record pertama hari ini
	LastUpdatedAt   time.Time `firestore:"lastUpdatedAt"`   // Waktu update terakhir
}

var (
	realtimeDBClient *db.Client
	firestoreClient  *firestore.Client
)

const (
	serviceAccountPath   = "serviceAccountKey.json"
	realtimeDatabaseURL  = "https://smarthome-a3fc2-default-rtdb.firebaseio.com"
	checkIntervalSeconds = 10 * time.Second // Interval pembacaan data
	firestoreCollection  = "electricity_history"
)

func main() {
	ctx := context.Background()
	if err := initializeFirebase(ctx); err != nil {
		log.Fatalf("❌ Error initializing Firebase: %v", err)
	}
	defer firestoreClient.Close()
	printStartupMessage()
	monitorElectricity(ctx)
}

func initializeFirebase(ctx context.Context) error {
	opt := option.WithCredentialsFile(serviceAccountPath)
	config := &firebase.Config{
		DatabaseURL: realtimeDatabaseURL,
	}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return fmt.Errorf("error initializing Firebase app: %v", err)
	}
	dbClient, err := app.Database(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Realtime Database: %v", err)
	}
	realtimeDBClient = dbClient
	fsClient, err := firestore.NewClient(ctx, "smarthome-a3fc2", opt)
	if err != nil {
		return fmt.Errorf("error initializing Firestore: %v", err)
	}
	firestoreClient = fsClient
	return nil
}

func getElectricityData(ctx context.Context) (*ListrikData, error) {
	var data ListrikData
	ref := realtimeDBClient.NewRef("IoTSystem/Listrik")
	if err := ref.Get(ctx, &data); err != nil {
		return nil, fmt.Errorf("error getting electricity data: %v", err)
	}
	return &data, nil
}

func saveToFirestore(ctx context.Context, data *ListrikData) error {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	docID := dateStr
	docRef := firestoreClient.Collection(firestoreCollection).Doc(docID)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		history := ElectricityHistory{
			TotalDaya_W:     data.DayaSaatIni_W,
			JumlahPembacaan: 1,
			RataRata_W:      data.DayaSaatIni_W,
			DayaTerakhir_W:  data.DayaSaatIni_W,
			Date:            dateStr,
			Year:            now.Year(),
			Month:           int(now.Month()),
			Day:             now.Day(),
			FirstRecordedAt: now,
			LastUpdatedAt:   now,
		}
		_, err := docRef.Set(ctx, history)
		if err != nil {
			return fmt.Errorf("error creating new document: %v", err)
		}
		log.Printf("📝 Dokumen baru dibuat untuk tanggal %s", dateStr)
		return nil
	}
	if docSnap.Exists() {
		var existingData ElectricityHistory
		if err := docSnap.DataTo(&existingData); err != nil {
			return fmt.Errorf("error reading existing document: %v", err)
		}
		newTotal := existingData.TotalDaya_W + data.DayaSaatIni_W
		newCount := existingData.JumlahPembacaan + 1
		newAverage := newTotal / float64(newCount)
		updates := []firestore.Update{
			{Path: "totalDaya_W", Value: newTotal},
			{Path: "jumlahPembacaan", Value: newCount},
			{Path: "rataRata_W", Value: newAverage},
			{Path: "dayaTerakhir_W", Value: data.DayaSaatIni_W},
			{Path: "lastUpdatedAt", Value: now},
		}
		_, err := docRef.Update(ctx, updates)
		if err != nil {
			return fmt.Errorf("error updating document: %v", err)
		}
		return nil
	}
	return fmt.Errorf("unexpected state: document snapshot exists but not valid")
}

func monitorElectricity(ctx context.Context) {
	ticker := time.NewTicker(checkIntervalSeconds)
	defer ticker.Stop()
	fmt.Printf("🔌 Monitoring listrik dimulai (interval: %v)\n", checkIntervalSeconds)
	fmt.Println("📊 Data akan disimpan ke Firestore collection:", firestoreCollection)
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	recordCount := 0
	for {
		select {
		case <-ticker.C:
			data, err := getElectricityData(ctx)
			if err != nil {
				log.Printf("❌ Error getting data: %v", err)
				continue
			}
			if err := saveToFirestore(ctx, data); err != nil {
				log.Printf("❌ Error saving to Firestore: %v", err)
				continue
			}
			recordCount++
			dateStr := time.Now().Format("2006-01-02")
			docRef := firestoreClient.Collection(firestoreCollection).Doc(dateStr)
			docSnap, err := docRef.Get(ctx)
			if err == nil && docSnap.Exists() {
				var currentData ElectricityHistory
				docSnap.DataTo(&currentData)
				fmt.Printf("[%s] ✅ Update #%d | Daya Saat Ini: %.2f W | Total Hari Ini: %.2f W | Rata-rata: %.2f W | Pembacaan: %d kali\n",
					time.Now().Format("15:04:05"),
					recordCount,
					data.DayaSaatIni_W,
					currentData.TotalDaya_W,
					currentData.RataRata_W,
					currentData.JumlahPembacaan)
			} else {
				fmt.Printf("[%s] ✅ Update #%d | Daya: %.2f W | Saved to Firestore\n",
					time.Now().Format("15:04:05"),
					recordCount,
					data.DayaSaatIni_W)
			}
		case <-ctx.Done():
			fmt.Println("\n⚠️  Context cancelled, stopping monitoring...")
			return
		}
	}
}

func printStartupMessage() {
	lines := []string{
		"",
		"==================== Monitoring Listrik ====================",
		"",
		"📊 Smart Home Electricity History Logger",
		"",
		"✓ Realtime Database: Connected",
		"✓ Firestore: Connected",
		fmt.Sprintf("✓ Interval: %v", checkIntervalSeconds),
		fmt.Sprintf("✓ Collection: %s", firestoreCollection),
		"",
		"Server menyala ..........................................",
		"",
		strings.Repeat("=", 60),
		"",
	}
	for _, line := range lines {
		fmt.Println(line)
		time.Sleep(100 * time.Millisecond)
	}
}
