package id

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// crockfordAlphabet is the Crockford base32 encoding alphabet.
const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// crockfordDecode maps each valid input character to its 5-bit value.
// Supports case-insensitive decode plus i/l → 1, o → 0 per Crockford.
var crockfordDecode [256]int8

func init() {
	for i := range crockfordDecode {
		crockfordDecode[i] = -1
	}
	for i, c := range crockfordAlphabet {
		crockfordDecode[c] = int8(i)
		if c >= 'A' && c <= 'Z' {
			crockfordDecode[c+32] = int8(i) // lowercase
		}
	}
	// Crockford mappings for visually ambiguous characters
	crockfordDecode['i'] = 1
	crockfordDecode['I'] = 1
	crockfordDecode['l'] = 1
	crockfordDecode['L'] = 1
	crockfordDecode['o'] = 0
	crockfordDecode['O'] = 0
}

// tsidNow is the time function used by TSID generation. Overridable for testing.
var tsidNow = func() time.Time { return time.Now() }

// tsidRandInt generates a cryptographically secure random integer in [0, max).
// Overridable for testing.
var tsidRandInt = func(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

// GenerateTSID13 produces a new 13-character time-sorted ID.
// The first 10 characters encode a 48-bit millisecond timestamp.
// The last 3 characters encode a 15-bit random value.
func GenerateTSID13() (string, error) {
	ms := tsidNow().UnixMilli()
	if ms < 0 {
		return "", fmt.Errorf("timestamp must be non-negative")
	}

	randomVal, err := tsidRandInt(1 << 15) // 0..32767
	if err != nil {
		return "", fmt.Errorf("generate random component: %w", err)
	}

	// Encode: 10 chars for 48-bit timestamp + 3 chars for 15-bit random
	// Total encoded bits = 10*5 = 50 bits for timestamp (top 2 unused, set to 0)
	// and 3*5 = 15 bits for random.
	var buf [13]byte

	// Encode timestamp into first 10 characters (most significant first)
	ts := uint64(ms)
	for i := 9; i >= 0; i-- {
		buf[i] = crockfordAlphabet[ts&0x1F]
		ts >>= 5
	}

	// Encode random into last 3 characters
	rv := uint64(randomVal)
	for i := 12; i >= 10; i-- {
		buf[i] = crockfordAlphabet[rv&0x1F]
		rv >>= 5
	}

	return string(buf[:]), nil
}

// ValidateTSID13 checks if a string is a valid 13-character Crockford base32 value.
// It does NOT normalize case — use NormalizeTSID for that.
func ValidateTSID13(s string) error {
	if len(s) != 13 {
		return fmt.Errorf("TSID must be exactly 13 characters, got %d", len(s))
	}
	for i, c := range s {
		if c > 255 || crockfordDecode[c] < 0 {
			return fmt.Errorf("invalid Crockford base32 character %q at position %d", c, i)
		}
	}
	return nil
}

// NormalizeTSID converts a TSID string to canonical Crockford base32 form.
// It validates the input, uppercases, and substitutes ambiguous characters
// (I/L → 1, O → 0) per the Crockford base32 spec.
func NormalizeTSID(s string) (string, error) {
	if err := ValidateTSID13(s); err != nil {
		return "", err
	}
	var buf [13]byte
	for i := 0; i < 13; i++ {
		buf[i] = crockfordAlphabet[crockfordDecode[s[i]]]
	}
	return string(buf[:]), nil
}
