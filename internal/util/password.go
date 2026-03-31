package util

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateAPIKey 生成API密钥
func GenerateAPIKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	key := make([]byte, length)
	for i := range key {
		key[i] = charset[rng.Intn(len(charset))]
	}

	return string(key)
}
