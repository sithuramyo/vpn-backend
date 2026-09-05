package vpn

import "github.com/skip2/go-qrcode"

// GenerateQRPNG encodes the given payload (an ss:// URI) as a PNG QR code.
// The QR only ever carries what the backend already returns from
// GetAccessKeyConfig — never a raw secret outside that same payload.
func GenerateQRPNG(payload string, size int) ([]byte, error) {
	return qrcode.Encode(payload, qrcode.Medium, size)
}
