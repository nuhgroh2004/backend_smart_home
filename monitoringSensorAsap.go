package main

import (
	"context"
	"fmt"
	"log"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"google.golang.org/api/option"
)

type SensorAsapData struct {
	Nilai  int    `json:"nilai"`
	Status string `json:"status"`
}

type NotifikasiData struct {
	AsapTerdeteksi bool   `json:"asapTerdeteksi"`
	Pesan          string `json:"pesan"`
	TandonPenuh    bool   `json:"tandonPenuh"`
}

var (
	sensorAsapDBClient *db.Client
)

const (
	sensorAsapServiceAccountPath = "serviceAccountKey.json"
	sensorAsapDatabaseURL        = "https://smarthome-a3fc2-default-rtdb.firebaseio.com"
	sensorAsapCheckInterval      = 2 * time.Second
)

func RunMonitoringSensorAsap(ctx context.Context) {
	if err := initializeSensorAsapFirebase(ctx); err != nil {
		log.Printf("❌ Error initializing Firebase for sensor asap: %v", err)
		return
	}
	monitorSensorAsap(ctx)
}

func initializeSensorAsapFirebase(ctx context.Context) error {
	opt := option.WithCredentialsFile(sensorAsapServiceAccountPath)
	config := &firebase.Config{
		DatabaseURL: sensorAsapDatabaseURL,
	}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return fmt.Errorf("error initializing Firebase app: %v", err)
	}
	dbClient, err := app.Database(ctx)
	if err != nil {
		return fmt.Errorf("error initializing Realtime Database: %v", err)
	}
	sensorAsapDBClient = dbClient
	return nil
}

func getSensorAsapData(ctx context.Context) (*SensorAsapData, error) {
	var data SensorAsapData
	ref := sensorAsapDBClient.NewRef("IoTSystem/SensorAsap")
	if err := ref.Get(ctx, &data); err != nil {
		return nil, fmt.Errorf("error getting sensor asap data: %v", err)
	}
	return &data, nil
}

func updateNotifikasiAsap(ctx context.Context, asapTerdeteksi bool) error {
	ref := sensorAsapDBClient.NewRef("IoTSystem/Notifikasi")

	var currentNotif NotifikasiData
	if err := ref.Get(ctx, &currentNotif); err != nil {
		return fmt.Errorf("error getting notifikasi data: %v", err)
	}

	if currentNotif.AsapTerdeteksi != asapTerdeteksi {
		updates := map[string]interface{}{
			"asapTerdeteksi": asapTerdeteksi,
		}

		if asapTerdeteksi {
			updates["pesan"] = "⚠️ BAHAYA! Asap terdeteksi"
		} else {
			updates["pesan"] = "Semua sistem normal"
		}

		if err := ref.Update(ctx, updates); err != nil {
			return fmt.Errorf("error updating notifikasi: %v", err)
		}

		if asapTerdeteksi {
			log.Printf("🔥 Status asap berubah: BAHAYA terdeteksi!")
		} else {
			log.Printf("✅ Status asap berubah: Kembali AMAN")
		}
	}

	return nil
}

func monitorSensorAsap(ctx context.Context) {
	ticker := time.NewTicker(sensorAsapCheckInterval)
	defer ticker.Stop()

	var previousStatus string

	for {
		select {
		case <-ticker.C:
			data, err := getSensorAsapData(ctx)
			if err != nil {
				log.Printf("❌ Error getting sensor asap data: %v", err)
				continue
			}

			if data.Status != previousStatus {
				fmt.Printf("[%s] 🔍 Sensor Asap | Nilai: %d | Status: %s\n",
					time.Now().Format("15:04:05"),
					data.Nilai,
					data.Status)
				previousStatus = data.Status
			}

			asapTerdeteksi := (data.Status == "BAHAYA")

			if err := updateNotifikasiAsap(ctx, asapTerdeteksi); err != nil {
				log.Printf("❌ Error updating notifikasi: %v", err)
				continue
			}

		case <-ctx.Done():
			return
		}
	}
}
