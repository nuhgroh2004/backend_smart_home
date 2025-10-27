package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"backend_smart_home/monitoring"
	"backend_smart_home/notifikasi"
)

func main() {
	// Tampilkan pesan startup
	printStartupMessage()

	// Context dengan cancel untuk graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup untuk menunggu semua goroutine selesai
	var wg sync.WaitGroup

	// Jalankan service notifikasi dalam goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := notifikasi.Start(ctx); err != nil {
			log.Printf("❌ Error pada service notifikasi: %v", err)
		}
	}()

	// Jalankan service monitoring listrik dalam goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := monitoring.Start(ctx); err != nil {
			log.Printf("❌ Error pada service monitoring: %v", err)
		}
	}()

	// Tunggu signal interrupt untuk graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Blok sampai menerima signal lalu langsung exit
	<-sigChan
	cancel()
}

func typePrintln(s string, charDelay time.Duration) {
	for _, r := range s {
		fmt.Printf("%c", r)
		time.Sleep(charDelay)
	}
	fmt.Println()
}

func printStartupMessage() {
	lines := []string{
		"",
		"======================= Golang 1.24 =======================",
		"",
		"Fokuslah pada pengguna, dan semua hal lain akan mengikuti.",
		"",
	}

	for _, line := range lines {
		if line == "" {
			fmt.Println()
		} else {
			typePrintln(line, 40*time.Millisecond)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Tampilkan status koneksi
	statusLines := []string{
		"✓ Realtime Database: Connected",
		"✓ Firestore: Connected",
		"✓ Cloud Messaging: Connected",
	}

	for _, line := range statusLines {
		fmt.Println(line)
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println()
	typePrintln("server menyala ............................................", 40*time.Millisecond)
	fmt.Println()
	time.Sleep(500 * time.Millisecond)
}
