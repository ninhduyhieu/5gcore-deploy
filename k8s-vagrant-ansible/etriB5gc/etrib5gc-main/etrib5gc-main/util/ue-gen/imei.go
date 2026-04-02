package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// GenerateIdentifier generates either an IMEI (15-digit) or
// IMEISV (16-digit) identifier.
func randomImei(isImei bool) string {
	rand.Seed(time.Now().UnixNano())

	// Generate 14-digit base: TAC (8) + SNR (6)
	base := ""
	for i := 0; i < 14; i++ {
		base += strconv.Itoa(rand.Intn(10))
	}

	if isImei {
		checkDigit := calculateLuhnDigit(base)
		return base + strconv.Itoa(checkDigit)
	} else {
		svn := fmt.Sprintf("%02d", rand.Intn(100))
		return base + svn
	}
}

func calculateLuhnDigit(base string) int {
	sum := 0
	for i := 0; i < len(base); i++ {
		digit := int(base[i] - '0')
		if i%2 == 1 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return (10 - (sum % 10)) % 10
}
