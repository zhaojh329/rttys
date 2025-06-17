package utils

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/google/uuid"
)

// GenUniqueID generate a unique ID
func GenUniqueID() string {
	hash := md5.Sum([]byte(uuid.New().String()))
	return hex.EncodeToString(hash[:16])
}
