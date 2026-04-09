package uecontext

import (
	"github.com/reogac/nas"
	"github.com/warthog618/sms/encoding/gsm7"
	"time"
)

func nasNetworkName(s string) *nas.NetworkName {
	u, err := gsm7.Encode([]byte(s))
	if err != nil {
		return nil
	}
	buf := gsm7.Pack7Bit(u, 0)

	ie := &nas.NetworkName{
		Bytes: make([]byte, len(buf)+1),
	}

	copy(ie.Bytes[1:], buf)
	return ie
}

func encodeUniversalTimeAndLocalTimeZone(t time.Time) (v [7]byte) {
	// Convert to local time first
	local := t.Local()
	_, offset := local.Zone() // offset in seconds

	// --- Encode Time ---
	year := local.Year() % 100
	month := int(local.Month())
	day := local.Day()
	hour := local.Hour()
	minute := local.Minute()
	second := local.Second()

	v[0] = toBCD(uint8(year))
	v[1] = toBCD(uint8(month))
	v[2] = toBCD(uint8(day))
	v[3] = toBCD(uint8(hour))
	v[4] = toBCD(uint8(minute))
	v[5] = toBCD(uint8(second))

	// Encode Time Zone/ (1/4 hour units)
	quarters := uint8(abs(offset) / 900) // 900s = 15m
	tz := toBCD(quarters)
	if offset < 0 {
		tz |= 0x08 << 4 // set sign bit (bit 3 of high nibble)
	}
	v[6] = tz
	return
}

//int to BCD
func toBCD(n uint8) uint8 {
	return ((n / 10) << 4) | (n % 10)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
