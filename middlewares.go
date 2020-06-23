package main

import (
	"net/http"
	"fmt"
	"github.com/gin-gonic/gin"

	jwt "github.com/dgrijalva/jwt-go"
)


type Claims struct {
	Userid int `json:"user_id"`
	jwt.StandardClaims
}

// generate ssh keys using keygen as described in: https://blog.miguelgrinberg.com/post/json-web-tokens-with-public-key-signatures
// put publine key should be in pem format, to convert .pub file generate above into .pem file use https://unix.stackexchange.com/a/310350/332228
func verifyToken(token string) (*Claims, error) {
	keys := km.GetKeyList()

    for i, key := range keys {
		claims := &Claims{}
		tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			return key, nil
		})
		
		// if err != nil {
		// 	return claims, err
		// }
		// if !tkn.Valid {
		// 	return claims, fmt.Errorf("Token Invalid")
		// }
		if err == nil && tkn.Valid {
			return claims, nil
		}
		fmt.Println(i)
	}

	return nil, fmt.Errorf("Token is invalid")
}

func AuthRequiredMiddleware(c *gin.Context) {
	authToken := c.GetHeader("X-Auth-Token")

	fmt.Println("checking if valid token...")
	claims, err := verifyToken(authToken)
    if err != nil && err.Error() == "Token is invalid" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"status": "error",
			"data": gin.H{
				"message": "You must be logged in to do this",
			},
		})
		return
    }

	c.Set("userId", claims.Userid)

	// Pass on to the next-in-chain
	c.Next()
}
