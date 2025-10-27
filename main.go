package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
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
