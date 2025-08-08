package main

import (
	"gcsuploader/handler"
	"gcsuploader/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	port := "8080"

	if err := handler.ConnectGCS("config/credentials.json", "dmtfota"); err != nil {
		log.Fatalf("Failed to connect to GCS: %v", err)
	}

	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	routes.GCSRouter(router)
	router.Run(":" + port)
}
