package util

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"strings"

	logger "github.com/phil-inc/plog-ng/pkg/core"

	"golang.org/x/crypto/bcrypt"
)

var iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func decodeBase64(s string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func Encrypt(key, text string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	plaintext := []byte(text)
	cfb := cipher.NewCFBEncrypter(block, iv)
	cipherText := make([]byte, len(plaintext))
	cfb.XORKeyStream(cipherText, plaintext)
	return encodeBase64(cipherText), nil
}

func Decrypt(key, text string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	cipherText, err := decodeBase64(text)
	if err != nil {
		return "", err
	}
	cfb := cipher.NewCFBDecrypter(block, iv)
	plaintext := make([]byte, len(cipherText))
	cfb.XORKeyStream(plaintext, cipherText)
	return string(plaintext), nil
}

func GetEncryptedUserPassword(rawPassword string) string {
	pb := []byte(rawPassword)
	// Hashing the password with the default cost of 10
	hashedPassword, hashError := bcrypt.GenerateFromPassword(pb, bcrypt.DefaultCost)
	if hashError != nil {
		logger.Errorf("Password encryption failed. Err: %s", hashError)
		return ""
	}
	return string(hashedPassword)
}

func urlEncodeBase64(b []byte) string {
	return base64.URLEncoding.EncodeToString(b)
}
func urlDecodeBase64(s string) ([]byte, error) {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func UrlEncrypt(key, text string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	plaintext := []byte(text)
	cfb := cipher.NewCFBEncrypter(block, iv)
	cipherText := make([]byte, len(plaintext))
	cfb.XORKeyStream(cipherText, plaintext)
	return urlEncodeBase64(cipherText), nil
}

func UrlDecrypt(key, text string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	//Replace character to handle legacy tokens.
	parsedText := strings.ReplaceAll(text, "+", "-")
	parsedText = strings.ReplaceAll(parsedText, "/", "_")

	cipherText, err := urlDecodeBase64(parsedText)
	if err != nil {
		return "", err
	}
	cfb := cipher.NewCFBDecrypter(block, iv)
	plaintext := make([]byte, len(cipherText))
	cfb.XORKeyStream(plaintext, cipherText)
	return string(plaintext), nil
}
