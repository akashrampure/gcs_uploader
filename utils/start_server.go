package utils

import (
	"gcsuploader/handler"
	"gcsuploader/routes"
	"log"

	"github.com/gin-gonic/gin"
)

func Start(port, credentialsPath, bucketName string) {

	if err := handler.ConnectGCS(credentialsPath, bucketName); err != nil {
		log.Fatalf("Failed to connect to GCS: %v", err)
	}

	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	routes.GCSRouter(router)
	router.Run(":" + port)
}
