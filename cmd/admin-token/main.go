package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "generate" {
		token, err := generateSecureToken()
		if err != nil {
			fmt.Printf("Error generating token: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("🔐 ADMIN SETUP TOKEN GENERATED")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()
		fmt.Println("Add these lines to your .env file:")
		fmt.Println()
		fmt.Printf("ADMIN_SETUP_TOKEN=%s\n", token)
		fmt.Printf("ADMIN_EMAIL=admin@bitcoinpitch.org\n")
		fmt.Printf("SITE_URL=http://localhost:8090\n")
		fmt.Println()
		fmt.Println("Optional SMTP settings (for email registration):")
		fmt.Printf("SMTP_HOST=your-smtp-server.com\n")
		fmt.Printf("SMTP_PORT=587\n")
		fmt.Printf("SMTP_USERNAME=your-smtp-username\n")
		fmt.Printf("SMTP_PASSWORD=your-smtp-password\n")
		fmt.Printf("SMTP_FROM_EMAIL=noreply@bitcoinpitch.org\n")
		fmt.Printf("SMTP_FROM_NAME=BitcoinPitch.org\n")
		fmt.Println()
		fmt.Println("⚠️  IMPORTANT:")
		fmt.Println("- This token will be the admin password")
		fmt.Println("- Change the password after first login")
		fmt.Println("- Enable TOTP 2FA for enhanced security")
		fmt.Println("- Keep this token secure!")
		fmt.Println()
		fmt.Println(strings.Repeat("=", 60))
		return
	}

	fmt.Println("Admin Token Generator")
	fmt.Println("Usage: go run cmd/admin-token/main.go generate")
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
