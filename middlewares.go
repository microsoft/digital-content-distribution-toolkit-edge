// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	ini "gopkg.in/ini.v1"

	jwt "github.com/dgrijalva/jwt-go"
)

type Claims struct {
	Userid string `json:"user_id"`
	jwt.StandardClaims
}

var middlewareCfg *ini.File = nil

// generate ssh keys using keygen as described in: https://blog.miguelgrinberg.com/post/json-web-tokens-with-public-key-signatures
// put publine key should be in pem format, to convert .pub file generate above into .pem file use https://unix.stackexchange.com/a/310350/332228
func verifyToken(token string) (*Claims, error) {
	keys := km.GetKeyList()
	fmt.Println("Length of keys: ", len(keys))
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
		fmt.Println(err.Error())
		log.Println(err.Error())
		fmt.Println(i)
		logger.Log_old("Debug", "Middleware", map[string]string{"Token": token, "Function": "verifyToken", "Error": err.Error(), "publicKey": string(i)})
	}
	return nil, fmt.Errorf("Token is invalid")
}

func AuthRequiredMiddleware(c *gin.Context) {
	logger.Log_old("Debug", "Middleware", map[string]string{"Request path": c.FullPath(), "Request queries": ""})
	authToken := c.GetHeader("Authorization")
	fmt.Println("Received token is: ", authToken)
	authToken = strings.Replace(authToken, "Bearer ", "", 1)
	fmt.Println("Relevant part: ", authToken)
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
	c.Header("hubId", device_cfg.Section("DEVICE_DETAIL").Key("deviceId").String())
	// Pass on to the next-in-chain
	c.Next()
}
