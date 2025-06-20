package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil/bech32"
)

// NostrEvent represents a Nostr event structure
type NostrEvent struct {
	ID        string     `json:"id"`
	PubKey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      int        `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig"`
}

// ValidateNostrPubkey validates if a string is a valid Nostr public key
func ValidateNostrPubkey(pubkey string) error {
	// Nostr pubkeys are 64-character hex strings
	if len(pubkey) != 64 {
		return errors.New("pubkey must be 64 characters long")
	}

	// Check if it's valid hex
	if _, err := hex.DecodeString(pubkey); err != nil {
		return errors.New("pubkey must be valid hex")
	}

	return nil
}

// ValidateNostrNpub validates if a string is a valid Nostr npub
func ValidateNostrNpub(npub string) error {
	// Basic format check: npub1 followed by 58 characters
	npubRegex := regexp.MustCompile(`^npub1[a-zA-Z0-9]{58}$`)
	if !npubRegex.MatchString(npub) {
		return errors.New("invalid npub format")
	}
	return nil
}

// VerifyNostrEvent verifies a Nostr event signature
func VerifyNostrEvent(event map[string]interface{}) error {
	// Convert to our struct for easier handling
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	var nostrEvent NostrEvent
	if err := json.Unmarshal(eventBytes, &nostrEvent); err != nil {
		return fmt.Errorf("failed to unmarshal event: %v", err)
	}

	// Validate required fields
	if nostrEvent.PubKey == "" {
		return errors.New("missing pubkey")
	}
	if nostrEvent.Sig == "" {
		return errors.New("missing signature")
	}

	// Validate pubkey format
	if err := ValidateNostrPubkey(nostrEvent.PubKey); err != nil {
		return fmt.Errorf("invalid pubkey: %v", err)
	}

	// Create the event ID (hash of serialized event)
	eventID := createNostrEventID(nostrEvent)

	// If ID is provided, verify it matches
	if nostrEvent.ID != "" && nostrEvent.ID != eventID {
		return errors.New("event ID mismatch")
	}

	// Verify the signature
	return verifyNostrSignature(eventID, nostrEvent.Sig, nostrEvent.PubKey)
}

// createNostrEventID creates the event ID by hashing the serialized event
func createNostrEventID(event NostrEvent) string {
	// Create serialized event array according to NIP-01
	serialized := []interface{}{
		0, // reserved for future use
		event.PubKey,
		event.CreatedAt,
		event.Kind,
		event.Tags,
		event.Content,
	}

	// Convert to JSON
	jsonBytes, _ := json.Marshal(serialized)

	// SHA256 hash
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// verifyNostrSignature verifies a Schnorr signature for a Nostr event
func verifyNostrSignature(eventID, signature, pubkey string) error {
	// Decode the event ID (32 bytes)
	eventIDBytes, err := hex.DecodeString(eventID)
	if err != nil {
		return fmt.Errorf("invalid event ID hex: %v", err)
	}
	if len(eventIDBytes) != 32 {
		return errors.New("event ID must be 32 bytes")
	}

	// Decode the signature (64 bytes)
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature hex: %v", err)
	}
	if len(sigBytes) != 64 {
		return errors.New("signature must be 64 bytes")
	}

	// Decode the public key (32 bytes)
	pubkeyBytes, err := hex.DecodeString(pubkey)
	if err != nil {
		return fmt.Errorf("invalid pubkey hex: %v", err)
	}
	if len(pubkeyBytes) != 32 {
		return errors.New("pubkey must be 32 bytes")
	}

	// Parse the public key
	pubKey, err := schnorr.ParsePubKey(pubkeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}

	// Parse the signature
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %v", err)
	}

	// Verify the signature
	if !sig.Verify(eventIDBytes, pubKey) {
		return errors.New("signature verification failed")
	}

	return nil
}

// ExtractPubkeyFromEvent extracts the public key from a Nostr event
func ExtractPubkeyFromEvent(event map[string]interface{}) (string, error) {
	pubkey, ok := event["pubkey"].(string)
	if !ok || pubkey == "" {
		return "", errors.New("missing or invalid pubkey in event")
	}

	if err := ValidateNostrPubkey(pubkey); err != nil {
		return "", fmt.Errorf("invalid pubkey: %v", err)
	}

	return pubkey, nil
}

// GenerateNostrDisplayName creates a readable display name from a Nostr pubkey
func GenerateNostrDisplayName(pubkey string) string {
	// Convert to npub format and show first 12 characters
	npub, err := HexToNpub(pubkey)
	if err != nil {
		// Fallback to old format if conversion fails
		if len(pubkey) >= 8 {
			return "nostr_" + pubkey[:8]
		}
		return "nostr_user"
	}

	// Show first 12 characters of npub (npub1 + 8 chars)
	if len(npub) >= 12 {
		return npub[:12] + "..."
	}
	return npub
}

// GenerateNostrUsername creates a unique username from a Nostr pubkey
func GenerateNostrUsername(pubkey string) string {
	// Convert to npub format for username
	npub, err := HexToNpub(pubkey)
	if err != nil {
		// Fallback to first 12 characters of pubkey for username uniqueness
		if len(pubkey) >= 12 {
			return pubkey[:12]
		}
		return pubkey
	}
	return npub
}

// HexToNpub converts a hex pubkey to npub (bech32) format
func HexToNpub(hexPubkey string) (string, error) {
	// Decode hex pubkey
	pubkeyBytes, err := hex.DecodeString(hexPubkey)
	if err != nil {
		return "", fmt.Errorf("invalid hex pubkey: %v", err)
	}

	if len(pubkeyBytes) != 32 {
		return "", errors.New("pubkey must be 32 bytes")
	}

	// Convert to bech32 with "npub" prefix
	converted, err := bech32.ConvertBits(pubkeyBytes, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %v", err)
	}

	npub, err := bech32.Encode("npub", converted)
	if err != nil {
		return "", fmt.Errorf("failed to encode bech32: %v", err)
	}

	return npub, nil
}

// GetFullNpub returns the full npub for use in tooltips or full display
func GetFullNpub(pubkey string) string {
	npub, err := HexToNpub(pubkey)
	if err != nil {
		return pubkey // fallback to hex if conversion fails
	}
	return npub
}
