package api

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Authenticator 校验单用户登录密码并签发/校验有时效 token。
// 密码以 env 注入明文，启动时 bcrypt 哈希到内存（明文不落库/不落文件）。
type Authenticator struct {
	passwordHash []byte
	secret       []byte
	tokenTTL     time.Duration
}

// NewAuthenticator 用明文密码（env 注入）和签名密钥构造。
func NewAuthenticator(plainPassword, secret string) (*Authenticator, error) {
	if plainPassword == "" || secret == "" {
		return nil, errors.New("password 和 secret 不能为空")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Authenticator{passwordHash: hash, secret: []byte(secret), tokenTTL: 24 * time.Hour}, nil
}

// CheckPassword 用 bcrypt 常量时间比较登录密码。
func (a *Authenticator) CheckPassword(pw string) bool {
	return bcrypt.CompareHashAndPassword(a.passwordHash, []byte(pw)) == nil
}

// IssueToken 签发一个带过期时间的 HS256 token（now 作参数便于测试）。
func (a *Authenticator) IssueToken(now time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(a.tokenTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
		Subject:   "dashboard",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// ValidateToken 校验 token 签名与有效期。
func (a *Authenticator) ValidateToken(tokenStr string) error {
	_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	return err
}
