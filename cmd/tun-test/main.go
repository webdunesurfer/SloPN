package main

import (
	"fmt"
	"log"
	"time"

	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

func main() {
	fmt.Println("Starting TUN test on Windows...")

	cfg := tunutil.Config{
		Name: "slopn-tap0",
		Addr: "10.100.0.99",
		Mask: "255.255.255.0",
		MTU:  1280,
	}

	ifce, err := tunutil.CreateInterface(cfg)
	if err != nil {
		log.Fatalf("Failed to create TUN: %v", err)
	}
	defer ifce.Close()

	fmt.Printf("Interface %s created successfully!\n", ifce.Name())
	fmt.Println("You can now check 'ipconfig' or 'netsh interface ip show config'.")
	fmt.Println("Sleeping for 30 seconds...")
	time.Sleep(30 * time.Second)
	fmt.Println("Closing interface and exiting.")
}