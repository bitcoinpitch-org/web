package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
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
func VerifyNostrEvent(event interface{}) error {
	eventMap := event.(map[string]interface{})

	// Extract required fields
	pubkeyStr, ok := eventMap["pubkey"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid pubkey")
	}

	signatureStr, ok := eventMap["sig"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid signature")
	}

	// Parse signature
	sigBytes, err := hex.DecodeString(signatureStr)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %v", err)
	}

	// Parse signature as Schnorr (64 bytes for Nostr)
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %v", err)
	}

	// Parse pubkey (32 bytes for x-only pubkey)
	pubkeyBytes, err := hex.DecodeString(pubkeyStr)
	if err != nil {
		return fmt.Errorf("failed to decode pubkey: %v", err)
	}

	if len(pubkeyBytes) != 32 {
		return fmt.Errorf("malformed public key: invalid length: %d", len(pubkeyBytes))
	}

	// Parse as x-only public key (Schnorr format)
	pubKey, err := schnorr.ParsePubKey(pubkeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}

	// Create event serialization for verification
	eventData := []interface{}{
		0,
		eventMap["pubkey"],
		eventMap["created_at"],
		eventMap["kind"],
		eventMap["tags"],
		eventMap["content"],
	}

	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %v", err)
	}

	eventHash := sha256.Sum256(eventBytes)

	// Verify signature
	if !sig.Verify(eventHash[:], pubKey) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
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
	// Use first 12 characters with npub prefix
	if len(pubkey) >= 12 {
		return "npub1" + pubkey[:12] + "..."
	}
	return "npub1" + pubkey + "..."
}

// GenerateNostrUsername creates a unique username from a Nostr pubkey
func GenerateNostrUsername(pubkey string) string {
	// Use first 8 characters of pubkey with nostr_ prefix
	if len(pubkey) >= 8 {
		return "nostr_" + pubkey[:8]
	}
	return "nostr_" + pubkey
}

// HexToNpub converts a hex pubkey to npub (bech32) format
func HexToNpub(hexPubkey string) (string, error) {
	// Decode hex
	pubkeyBytes, err := hex.DecodeString(hexPubkey)
	if err != nil {
		return "", fmt.Errorf("failed to decode hex: %v", err)
	}

	if len(pubkeyBytes) != 32 {
		return "", fmt.Errorf("pubkey must be 32 bytes")
	}

	// Convert to 5-bit for bech32
	converted, err := bech32.ConvertBits(pubkeyBytes, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %v", err)
	}

	// Encode as bech32
	npub, err := bech32.Encode("npub", converted)
	if err != nil {
		return "", fmt.Errorf("failed to encode bech32: %v", err)
	}

	return npub, nil
}

// NpubToHex converts a npub (bech32) format to hex format
func NpubToHex(npub string) (string, error) {
	// Decode bech32
	hrp, data, err := bech32.Decode(npub)
	if err != nil {
		return "", fmt.Errorf("failed to decode bech32: %v", err)
	}

	if hrp != "npub" {
		return "", fmt.Errorf("invalid hrp, expected 'npub', got '%s'", hrp)
	}

	// Convert from 5-bit to 8-bit
	converted, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %v", err)
	}

	if len(converted) != 32 {
		return "", fmt.Errorf("invalid pubkey length: %d", len(converted))
	}

	return hex.EncodeToString(converted), nil
}

// NsecToHex converts a nsec (bech32) format to hex format
func NsecToHex(nsec string) (string, error) {
	// Decode bech32
	hrp, data, err := bech32.Decode(nsec)
	if err != nil {
		return "", fmt.Errorf("failed to decode bech32: %v", err)
	}

	if hrp != "nsec" {
		return "", fmt.Errorf("invalid hrp, expected 'nsec', got '%s'", hrp)
	}

	// Convert from 5-bit to 8-bit
	converted, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %v", err)
	}

	if len(converted) != 32 {
		return "", fmt.Errorf("invalid private key length: %d", len(converted))
	}

	return hex.EncodeToString(converted), nil
}

// ProcessManualNostrAuth processes manual authentication with private key
func ProcessManualNostrAuth(privateKeyHex string) (pubkey string, signature string, timestamp int64, message string, err error) {
	// Validate hex format
	if len(privateKeyHex) != 64 {
		return "", "", 0, "", fmt.Errorf("private key must be exactly 64 hex characters")
	}

	// Parse hex private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("invalid hex format: %v", err)
	}

	if len(privateKeyBytes) != 32 {
		return "", "", 0, "", fmt.Errorf("private key must be 32 bytes")
	}

	// Derive public key from private key (Nostr uses x-only public keys)
	privKey, pubKey := btcec.PrivKeyFromBytes(privateKeyBytes)

	// Nostr uses x-only public keys (32 bytes, not 33)
	pubkeyBytes := pubKey.SerializeCompressed()
	if len(pubkeyBytes) != 33 {
		return "", "", 0, "", fmt.Errorf("invalid public key length")
	}

	// Remove the first byte (compression flag) to get x-only pubkey
	pubkey = hex.EncodeToString(pubkeyBytes[1:])

	// Create event data (timestamp for uniqueness)
	timestamp = time.Now().Unix()
	message = fmt.Sprintf("BitcoinPitch Authentication: %d", timestamp)

	// Create event data structure for signing (Nostr NIP-01 format)
	// This must match exactly what VerifyNostrEvent will reconstruct
	eventData := []interface{}{
		0,            // reserved
		pubkey,       // pubkey
		timestamp,    // created_at
		1,            // kind
		[][]string{}, // tags
		message,      // content
	}

	// Serialize event data for signing
	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("failed to serialize event: %v", err)
	}

	// Hash the serialized event (Nostr event ID)
	eventHash := sha256.Sum256(eventBytes)

	// Sign the hash using Schnorr signature (correct for Nostr)
	sig, err := schnorr.Sign(privKey, eventHash[:])
	if err != nil {
		return "", "", 0, "", fmt.Errorf("failed to create signature: %v", err)
	}

	// Convert signature to hex format (Schnorr signatures are 64 bytes)
	signature = hex.EncodeToString(sig.Serialize())

	return pubkey, signature, timestamp, message, nil
}

// GetFullNpub returns the full npub for use in tooltips or full display
func GetFullNpub(hexPubkey string) string {
	npub, err := HexToNpub(hexPubkey)
	if err != nil {
		return hexPubkey[:12] + "..." // fallback to hex
	}
	return npub
}
