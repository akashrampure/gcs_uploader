package routes

import (
	gcs "gcsuploader/handler"

	"github.com/gin-gonic/gin"
)

func GCSRouter(r *gin.Engine) {
	api := r.Group("/api/v1/gcs")
	{
		api.GET("/list", gcs.ListFiles)
		api.POST("/upload", gcs.UploadFile)
		api.GET("/download", gcs.DownloadFile)
		api.DELETE("/delete", gcs.DeleteObject)
		api.POST("/upload-buffer", gcs.UploadBuffer)
	}
}
