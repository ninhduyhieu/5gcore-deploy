package main

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func md5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func generateRandomMsisdn(length int) string {
	const digits = "0123456789"

	msisdn := make([]byte, length)
	for i := 0; i < length; i++ {
		msisdn[i] = digits[rand.Intn(len(digits))]
	}
	return string(msisdn)
}
