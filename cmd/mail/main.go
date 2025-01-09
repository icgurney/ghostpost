package main

import (
	"context"
	"log"
	"os"

	"ghostpost/internal/smtp"
	"ghostpost/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "2525"
	}

	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("BUCKET_NAME environment variable not set")
	}

	acceptDomains := []string{
		"ghostpost.sh",
		"xn--9q8hgh.ws", // 👻📮.ws punycode domain
	}

	client, err := storage.NewTigrisClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to create Tigris client: %v", err)
	}

	store := storage.NewTigrisStorage(client, bucketName)
	server := smtp.NewServer(":"+port, store, acceptDomains)

	log.Printf("SMTP server listening on port %s\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
