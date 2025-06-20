package handlers

// Helper function to validate Twitter handle
func isValidTwitterHandle(handle string) bool {
	// Twitter handles are 1-15 characters, alphanumeric + underscore
	// Must start with @
	return len(handle) >= 2 && len(handle) <= 16 && handle[0] == '@' && isValidTwitterUsername(handle[1:])
}

// Helper function to validate Twitter username
func isValidTwitterUsername(username string) bool {
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// Helper function to validate Nostr pubkey
func isValidNostrPubkey(pubkey string) bool {
	// Nostr pubkeys are npub1 followed by 58 base58 characters
	return len(pubkey) == 63 && pubkey[:5] == "npub1" && isValidBase58(pubkey[5:])
}

// Helper function to validate base58 string
func isValidBase58(s string) bool {
	for _, c := range s {
		if !((c >= '1' && c <= '9') || (c >= 'A' && c <= 'H') || (c >= 'J' && c <= 'N') || (c >= 'P' && c <= 'Z') || (c >= 'a' && c <= 'k') || (c >= 'm' && c <= 'z')) {
			return false
		}
	}
	return true
}

// Helper function to validate tag
func isValidTag(tag string) bool {
	// Tags are 1-50 characters, alphanumeric + underscore + hyphen
	if len(tag) < 1 || len(tag) > 50 {
		return false
	}
	for _, c := range tag {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
