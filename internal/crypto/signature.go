package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

// BitcoinMessagePrefix is the standard Bitcoin message signing prefix
const BitcoinMessagePrefix = "\x18Bitcoin Signed Message:\n"

// VerifyBitcoinMessage verifies a Bitcoin message signature
func VerifyBitcoinMessage(message, signature, address string) error {
	// Decode the signature from base64
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %v", err)
	}

	if len(sigBytes) != 65 {
		return errors.New("signature must be 65 bytes")
	}

	// Extract recovery flag and signature
	recoveryFlag := sigBytes[0]
	if recoveryFlag < 27 || recoveryFlag > 34 {
		return errors.New("invalid recovery flag")
	}

	// Convert recovery flag to standard format (0-3)
	isCompressed := recoveryFlag >= 31
	recoveryFlag = (recoveryFlag - 27) & 3

	// Create the message hash using Bitcoin's message signing format
	messageHash := createMessageHash(message)

	// Recover the public key
	pubKey, wasCompressed, err := ecdsa.RecoverCompact(sigBytes[1:], messageHash[:])
	if err != nil {
		return fmt.Errorf("failed to recover public key: %v", err)
	}

	// Check if compression matches
	if wasCompressed != isCompressed {
		return errors.New("compression flag mismatch")
	}

	// Convert public key to address
	var pubKeyAddress string
	if isCompressed {
		pubKeyAddress, err = pubKeyToCompressedAddress(pubKey)
	} else {
		pubKeyAddress, err = pubKeyToUncompressedAddress(pubKey)
	}
	if err != nil {
		return fmt.Errorf("failed to convert public key to address: %v", err)
	}

	// Verify address matches
	if pubKeyAddress != address {
		return fmt.Errorf("address mismatch: expected %s, got %s", address, pubKeyAddress)
	}

	return nil
}

// createMessageHash creates a hash of the message using Bitcoin's message signing format
func createMessageHash(message string) [32]byte {
	// Create the full message with Bitcoin prefix
	fullMessage := BitcoinMessagePrefix + fmt.Sprintf("%c", len(message)) + message

	// Double SHA256 hash
	firstHash := sha256.Sum256([]byte(fullMessage))
	return sha256.Sum256(firstHash[:])
}

// pubKeyToCompressedAddress converts a public key to a compressed Bitcoin address
func pubKeyToCompressedAddress(pubKey *btcec.PublicKey) (string, error) {
	// Serialize public key in compressed format
	pubKeyBytes := pubKey.SerializeCompressed()

	// Create address from public key hash
	pubKeyHash := btcutil.Hash160(pubKeyBytes)
	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}

// pubKeyToUncompressedAddress converts a public key to an uncompressed Bitcoin address
func pubKeyToUncompressedAddress(pubKey *btcec.PublicKey) (string, error) {
	// Serialize public key in uncompressed format
	pubKeyBytes := pubKey.SerializeUncompressed()

	// Create address from public key hash
	pubKeyHash := btcutil.Hash160(pubKeyBytes)
	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}

// ValidateBitcoinAddress validates if a string is a valid Bitcoin address
func ValidateBitcoinAddress(address string) error {
	_, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams)
	return err
}
