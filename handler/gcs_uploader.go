package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCSUploader struct {
	credentialsJSON []byte
	bucket          string

	storageClient *storage.Client
	bucketHandle  *storage.BucketHandle
}

func NewGCSUploader(credentialsJSON []byte, bucket string) *GCSUploader {
	return &GCSUploader{
		credentialsJSON: credentialsJSON,
		bucket:          bucket,

		storageClient: nil,
		bucketHandle:  nil,
	}
}

func (o *GCSUploader) Init() error {
	clientopts := option.WithCredentialsJSON(o.credentialsJSON)

	client, err := storage.NewClient(context.Background(), clientopts)
	if err != nil {
		log.Println("Storage new client:Err:", err)
		return err
	}

	bucketHandle := client.Bucket(o.bucket)

	o.storageClient = client
	o.bucketHandle = bucketHandle

	return nil
}

func (o *GCSUploader) UploadFile(ctx context.Context, file io.Reader, objectname string, writerChunkSize int, progressf func(int64)) (int64, error) {
	if o.bucketHandle == nil {
		return 0, fmt.Errorf("bucket handle is not initialized")
	}

	objectHandle := o.bucketHandle.Object(objectname)

	objectWriter := objectHandle.NewWriter(ctx)

	objectWriter.ProgressFunc = progressf
	objectWriter.ChunkSize = writerChunkSize

	nbytescopied, err := io.Copy(objectWriter, file)
	if err != nil {
		objectWriter.Close()
		return 0, fmt.Errorf("io.Copy: %w", err)
	}

	closeErr := objectWriter.Close()
	if closeErr != nil {
		return 0, fmt.Errorf("object close failed with :%v", closeErr)
	}

	return nbytescopied, nil
}

func (o *GCSUploader) UploadBuffer(ctx context.Context, filecontent []byte, objectname string, writerChunkSize int, progressf func(int64)) (int64, error) {
	if o.bucketHandle == nil {
		return 0, fmt.Errorf("bucket handle is not initialized")
	}

	objectHandle := o.bucketHandle.Object(objectname)

	objectWriter := objectHandle.NewWriter(ctx)

	objectWriter.ProgressFunc = progressf
	objectWriter.ChunkSize = writerChunkSize

	nbytescopied, err := io.Copy(objectWriter, bytes.NewReader(filecontent))
	if err != nil {
		objectWriter.Close()
		return 0, fmt.Errorf("io.Copy: %w", err)
	}

	closeErr := objectWriter.Close()
	if closeErr != nil {
		return 0, fmt.Errorf("object close failed with :%v", closeErr)
	}

	return nbytescopied, nil
}

func (o *GCSUploader) DownloadFile(ctx context.Context, objectname string, destination string) (int64, error) {
	if o.bucketHandle == nil {
		return 0, fmt.Errorf("bucket handle is not initialized")
	}

	objectHandle := o.bucketHandle.Object(objectname)

	objectReader, readererr := objectHandle.NewReader(ctx)
	if readererr != nil {
		return 0, readererr
	}
	defer objectReader.Close()

	filename := filepath.Join(destination, path.Base(objectname))

	outputfile, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer outputfile.Close()

	nbytescopied, err := io.Copy(outputfile, objectReader)
	if err != nil {
		return 0, err
	}

	return nbytescopied, nil
}

func (o *GCSUploader) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	if o.bucketHandle == nil {
		return nil, fmt.Errorf("bucket handle is not initialized")
	}

	var objectNames []string

	query := &storage.Query{Prefix: prefix}
	it := o.bucketHandle.Objects(ctx, query)

	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing objects: %w", err)
		}

		objectNames = append(objectNames, objAttrs.Name)
	}

	return objectNames, nil
}

func (o *GCSUploader) DeleteObject(ctx context.Context, objectName string) error {
	if o.bucketHandle == nil {
		return fmt.Errorf("bucket handle is not initialized")
	}

	objectHandle := o.bucketHandle.Object(objectName)
	if err := objectHandle.Delete(ctx); err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return fmt.Errorf("object %q does not exist", objectName)
		}
		return fmt.Errorf("failed to delete object %q: %w", objectName, err)
	}

	return nil
}

func (o *GCSUploader) GetObjectUrl(ctx context.Context, objectName string) (string, error) {
	if o.bucketHandle == nil {
		return "", fmt.Errorf("bucket handle is not initialized")
	}

	signedUrl, err := o.bucketHandle.SignedURL(objectName, &storage.SignedURLOptions{
		Method:  "GET",
		Headers: []string{"*"},
		Expires: time.Now().Add(time.Hour * 24),
	})
	if err != nil {
		return "", err
	}
	return signedUrl, nil
}
