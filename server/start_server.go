package server

import (
	"gcsuploader/handler"
	"gcsuploader/routes"
	"gcsuploader/utils"
	"log"

	"github.com/gin-gonic/gin"
)

func Start() {
	utils.LoadEnv()

	port := utils.GetEnv("PORT", "8080")
	credentialsPath := utils.GetEnv("CREDENTIALS", "credentials.json")
	bucketName := utils.GetEnv("BUCKET_NAME", "dmtfota")

	if err := handler.ConnectGCS(credentialsPath, bucketName); err != nil {
		log.Fatalf("Failed to connect to GCS: %v", err)
	}

	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	routes.GCSRouter(router)
	log.Printf("Server started on port %s", port)
	router.Run(":" + port)
}
