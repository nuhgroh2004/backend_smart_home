package main

import (
	"context"
	"fmt"
	"log"
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
	TotalDaya_W     float64   `firestore:"totalDaya_W"`
	JumlahPembacaan int       `firestore:"jumlahPembacaan"`
	RataRata_W      float64   `firestore:"rataRata_W"`
	DayaTerakhir_W  float64   `firestore:"dayaTerakhir_W"`
	Date            string    `firestore:"date"`
	Year            int       `firestore:"year"`
	Month           int       `firestore:"month"`
	Day             int       `firestore:"day"`
	FirstRecordedAt time.Time `firestore:"firstRecordedAt"`
	LastUpdatedAt   time.Time `firestore:"lastUpdatedAt"`
}

var (
	monitoringRealtimeDBClient *db.Client
	monitoringFirestoreClient  *firestore.Client
)

const (
	monitoringServiceAccountPath  = "serviceAccountKey.json"
	monitoringRealtimeDatabaseURL = "https://smarthome-a3fc2-default-rtdb.firebaseio.com"
	monitoringCheckInterval       = 10 * time.Second
	monitoringFirestoreCollection = "electricity_history"
)

func RunMonitoringListrik(ctx context.Context) {
	if err := initMonitoringFirebase(ctx); err != nil {
		log.Printf("❌ Error initializing Firebase for monitoring: %v", err)
		return
	}
	monitorElectricity(ctx)
}

func initMonitoringFirebase(ctx context.Context) error {
	opt := option.WithCredentialsFile(monitoringServiceAccountPath)
	config := &firebase.Config{DatabaseURL: monitoringRealtimeDatabaseURL}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return fmt.Errorf("error initializing Firebase app: %v", err)
	}
	dbClient, err := app.Database(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Realtime Database: %v", err)
	}
	monitoringRealtimeDBClient = dbClient
	fsClient, err := firestore.NewClient(ctx, "smarthome-a3fc2", opt)
	if err != nil {
		return fmt.Errorf("error initializing Firestore: %v", err)
	}
	monitoringFirestoreClient = fsClient
	return nil
}

func getElectricityData(ctx context.Context) (*ListrikData, error) {
	var data ListrikData
	ref := monitoringRealtimeDBClient.NewRef("IoTSystem/Listrik")
	if err := ref.Get(ctx, &data); err != nil {
		return nil, fmt.Errorf("error getting electricity data: %v", err)
	}
	return &data, nil
}

func saveToFirestore(ctx context.Context, data *ListrikData) error {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	docRef := monitoringFirestoreClient.Collection(monitoringFirestoreCollection).Doc(dateStr)
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
	return fmt.Errorf("unexpected state")
}

func monitorElectricity(ctx context.Context) {
	ticker := time.NewTicker(monitoringCheckInterval)
	defer ticker.Stop()
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
			docRef := monitoringFirestoreClient.Collection(monitoringFirestoreCollection).Doc(dateStr)
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
			return
		}
	}
}
