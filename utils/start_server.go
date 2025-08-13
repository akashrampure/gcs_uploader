package utils

import (
	"gcsuploader/handler"
	"gcsuploader/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func StartServer() {
	LoadEnv()

	port := GetEnv("PORT", "8080")
	credentialsPath := GetEnv("CREDENTIALS", "credentials.json")
	bucketName := GetEnv("BUCKET_NAME", "dmtfota")

	if err := handler.ConnectGCS(credentialsPath, bucketName); err != nil {
		log.Fatalf("Failed to connect to GCS: %v", err)
	}

	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	routes.GCSRouter(router)
	log.Printf("Server started on port %s", port)
	router.Run(":" + port)
}
