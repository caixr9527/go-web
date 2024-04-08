package token

import (
	"github.com/caixr9527/zorm"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

type JwtHandler struct {
	// jwt 算法
	Alg            string
	TimeOut        time.Duration
	TimeFunc       func() time.Time
	Key            string
	PrivateKey     string
	SendCookie     bool
	Authenticator  func(ctx *zorm.Context) (map[string]any, error)
	CookieName     string
	CookieMaxAge   int64
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
}
type JwtResponse struct {
	Token        string
	RefreshToken string
}

func (j *JwtHandler) LoginHandler(ctx *zorm.Context) (*JwtResponse, error) {
	authenticator, err := j.Authenticator(ctx)
	if err != nil {
		return nil, err
	}
	if j.Alg == "" {
		j.Alg = "HS256"
	}
	signingMethod := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(signingMethod)
	claims := token.Claims.(jwt.MapClaims)
	if authenticator != nil {
		for key, value := range authenticator {
			claims[key] = value
		}
	}
	if j.TimeFunc == nil {
		j.TimeFunc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFunc().Add(j.TimeOut)
	claims["exp"] = expire.Unix()
	claims["iat"] = j.TimeFunc().Unix()
	var tokenString string
	var tokenErr error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = token.SignedString(j.PrivateKey)

	} else {
		tokenString, tokenErr = token.SignedString(j.Key)
	}
	if tokenErr != nil {
		return nil, tokenErr
	}
	jr := &JwtResponse{
		Token: tokenString,
	}
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = "zorm_token"
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = expire.Unix() - j.TimeFunc().Unix()
		}
		ctx.SetCookie(j.CookieName, tokenString, int(j.CookieMaxAge), "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	// todo RefreshToken
	return jr, nil
}

func (j *JwtHandler) usingPublicKeyAlgo() bool {
	switch j.Alg {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}
