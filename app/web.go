package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CheckSecret returns gin middleware to verify a shared secret header
func (config *Config) CheckSecret() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.Join(c.Request.Header["Secret"], "") == config.Secret {
			c.Next()
		} else {
			c.String(403, "Incorrect shared secret")
			c.Abort()
		}
	}
}

func BuildWeb(config *Config) *gin.Engine {
	router := gin.Default()

	api := router.Group("/api")
	{
		api.Use(config.CheckSecret())
		// LIST
		api.GET("/records", func(c *gin.Context) {
			c.JSON(http.StatusOK, config.Records)
		})
		// CREATE
		api.POST("/records", func(c *gin.Context) {
			var json NamedRecord
			if c.BindJSON(&json) == nil {
				config.Records[json.Name] = json.Record
			}
		})
		// SHOW
		api.GET("/records/:address", func(c *gin.Context) {
			c.JSON(http.StatusOK, config.Records[c.Param("address")])
		})
		// UPDATE
		api.PUT("/records/:address", func(c *gin.Context) {
			var json string
			if c.BindJSON(&json) == nil {
				config.Records[c.Param("address")] = json
			}
		})
		// DESTROY
		api.DELETE("/records/:address", func(c *gin.Context) {
			delete(config.Records, c.Param("address"))
		})
	}

	return router
}
