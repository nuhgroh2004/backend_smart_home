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
	TotalDaya_Wh    float64   `firestore:"totalDaya_Wh"`
	TotalDaya_kWh   float64   `firestore:"totalDaya_kWh"`
	JumlahPembacaan int       `firestore:"jumlahPembacaan"`
	RataRata_W      float64   `firestore:"rataRata_W"`
	DayaTerakhir_W  float64   `firestore:"dayaTerakhir_W"`
	BiayaHarian_Rp  float64   `firestore:"biayaHarian_Rp"`
	Date            string    `firestore:"date"`
	Year            int       `firestore:"year"`
	Month           int       `firestore:"month"`
	Day             int       `firestore:"day"`
	FirstRecordedAt time.Time `firestore:"firstRecordedAt"`
	LastUpdatedAt   time.Time `firestore:"lastUpdatedAt"`
	LastPembacaanAt time.Time `firestore:"lastPembacaanAt"`
}

type MonthlyElectricityHistory struct {
	TotalDaya_Wh       float64   `firestore:"totalDaya_Wh"`
	TotalKonsumsi_kWh  float64   `firestore:"totalKonsumsi_kWh"`
	TotalBiaya_Rp      float64   `firestore:"totalBiaya_Rp"`
	RataRataHarian_kWh float64   `firestore:"rataRataHarian_kWh"`
	JumlahHari         int       `firestore:"jumlahHari"`
	Month              int       `firestore:"month"`
	Year               int       `firestore:"year"`
	LastUpdatedAt      time.Time `firestore:"lastUpdatedAt"`
}

var (
	realtimeDBClient *db.Client
	firestoreClient  *firestore.Client
)

const (
	serviceAccountPath   = "serviceAccountKey.json"
	realtimeDatabaseURL  = "https://smarthome-a3fc2-default-rtdb.firebaseio.com"
	checkIntervalSeconds = 10 * time.Second
	firestoreCollection  = "electricity_history"
	monthlyCollection    = "monthly_electricity_history"
	tarifPerKwh_Rp       = 1352.0
)

func RunMonitoringListrik(ctx context.Context) {
	if err := initializeFirebase(ctx); err != nil {
		log.Printf("❌ Error initializing Firebase for monitoring: %v", err)
		return
	}
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
			TotalDaya_Wh:    0,
			TotalDaya_kWh:   0,
			JumlahPembacaan: 1,
			RataRata_W:      data.DayaSaatIni_W,
			DayaTerakhir_W:  data.DayaSaatIni_W,
			BiayaHarian_Rp:  0,
			Date:            dateStr,
			Year:            now.Year(),
			Month:           int(now.Month()),
			Day:             now.Day(),
			FirstRecordedAt: now,
			LastUpdatedAt:   now,
			LastPembacaanAt: now,
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
		newCount := existingData.JumlahPembacaan + 1
		intervalJam := now.Sub(existingData.LastPembacaanAt).Hours()
		rataRataInterval := (existingData.DayaTerakhir_W + data.DayaSaatIni_W) / 2.0
		konsumsiWh := rataRataInterval * intervalJam
		newTotalDaya_Wh := existingData.TotalDaya_Wh + konsumsiWh
		newTotalDaya_kWh := newTotalDaya_Wh / 1000.0
		konsumsiBaru_kWh := konsumsiWh / 1000.0
		biayaBaru_Rp := konsumsiBaru_kWh * tarifPerKwh_Rp
		newBiayaHarian_Rp := existingData.BiayaHarian_Rp + biayaBaru_Rp
		totalDurasiJam := now.Sub(existingData.FirstRecordedAt).Hours()
		var newAverage float64
		if totalDurasiJam > 0 {
			newAverage = newTotalDaya_Wh / totalDurasiJam
		} else {
			newAverage = data.DayaSaatIni_W
		}
		updates := []firestore.Update{
			{Path: "totalDaya_Wh", Value: newTotalDaya_Wh},
			{Path: "totalDaya_kWh", Value: newTotalDaya_kWh},
			{Path: "jumlahPembacaan", Value: newCount},
			{Path: "rataRata_W", Value: newAverage},
			{Path: "dayaTerakhir_W", Value: data.DayaSaatIni_W},
			{Path: "biayaHarian_Rp", Value: newBiayaHarian_Rp},
			{Path: "lastUpdatedAt", Value: now},
			{Path: "lastPembacaanAt", Value: now},
		}
		_, err := docRef.Update(ctx, updates)
		if err != nil {
			return fmt.Errorf("error updating document: %v", err)
		}
		if err := updateMonthlyHistory(ctx, now.Year(), int(now.Month())); err != nil {
			log.Printf("⚠️ Warning: Error updating monthly history: %v", err)
		}
		return nil
	}
	return fmt.Errorf("unexpected state: document snapshot exists but not valid")
}

func updateMonthlyHistory(ctx context.Context, year, month int) error {
	docID := fmt.Sprintf("%d-%02d", year, month)
	docRef := firestoreClient.Collection(monthlyCollection).Doc(docID)
	totalWhBulanan, totalKonsumsi, totalBiaya, jumlahHari, err := calculateMonthlyTotals(ctx, year, month)
	if err != nil {
		return fmt.Errorf("error calculating monthly totals: %v", err)
	}
	rataRataHarian := 0.0
	if jumlahHari > 0 {
		rataRataHarian = totalKonsumsi / float64(jumlahHari)
	}
	now := time.Now()
	monthlyData := MonthlyElectricityHistory{
		TotalDaya_Wh:       totalWhBulanan,
		TotalKonsumsi_kWh:  totalKonsumsi,
		TotalBiaya_Rp:      totalBiaya,
		RataRataHarian_kWh: rataRataHarian,
		JumlahHari:         jumlahHari,
		Month:              month,
		Year:               year,
		LastUpdatedAt:      now,
	}
	_, err = docRef.Set(ctx, monthlyData)
	if err != nil {
		return fmt.Errorf("error setting monthly document: %v", err)
	}
	return nil
}

func calculateMonthlyTotals(ctx context.Context, year, month int) (float64, float64, float64, int, error) {
	startDate := fmt.Sprintf("%d-%02d-01", year, month)
	endDate := fmt.Sprintf("%d-%02d-31", year, month)
	query := firestoreClient.Collection(firestoreCollection).
		OrderBy(firestore.DocumentID, firestore.Asc).
		StartAt(startDate).
		EndAt(endDate)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	var totalKonsumsiWh, totalKonsumsi, totalBiaya float64
	jumlahHari := 0
	for _, doc := range docs {
		var dailyData ElectricityHistory
		if err := doc.DataTo(&dailyData); err != nil {
			continue
		}
		if dailyData.Year == year && dailyData.Month == month {
			totalKonsumsiWh += dailyData.TotalDaya_Wh
			totalKonsumsi += dailyData.TotalDaya_kWh
			totalBiaya += dailyData.BiayaHarian_Rp
			jumlahHari++
		}
	}
	if jumlahHari == 0 {
		log.Printf("⚠️ Tidak ada data harian untuk bulan %d-%02d", year, month)
	}
	return totalKonsumsiWh, totalKonsumsi, totalBiaya, jumlahHari, nil
}

func monitorElectricity(ctx context.Context) {
	ticker := time.NewTicker(checkIntervalSeconds)
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
			docRef := firestoreClient.Collection(firestoreCollection).Doc(dateStr)
			docSnap, err := docRef.Get(ctx)
			if err == nil && docSnap.Exists() {
				var currentData ElectricityHistory
				if err := docSnap.DataTo(&currentData); err != nil {
					log.Printf("⚠️ Warning: Error reading document data: %v", err)
					fmt.Printf("[%s] ✅ Update #%d | Daya: %.2f W | Saved to Firestore\n",
						time.Now().Format("15:04:05"),
						recordCount,
						data.DayaSaatIni_W)
				} else {
					fmt.Printf("[%s] ✅ Update #%d | Daya: %.2f W | Total: %.2f Wh | Rata-rata: %.2f W | Konsumsi: %.4f kWh | Biaya: Rp %.2f | Pembacaan: %d kali\n",
						time.Now().Format("15:04:05"),
						recordCount,
						data.DayaSaatIni_W,
						currentData.TotalDaya_Wh,
						currentData.RataRata_W,
						currentData.TotalDaya_kWh,
						currentData.BiayaHarian_Rp,
						currentData.JumlahPembacaan)
				}
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
