package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/storage/models"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/filesystem"
	"github.com/darlingson/Oort-Object-Storage/internal/config"
)

var ErrBucketNotFound = errors.New("bucket not found")

type ObjectService struct {
	buckets repositories.BucketRepository
	objects repositories.ObjectRepository
	storage filesystem.StorageDriver
}

func NewObjectService(
	buckets repositories.BucketRepository,
	objects repositories.ObjectRepository,
	storage filesystem.StorageDriver,
) *ObjectService {
	return &ObjectService{
		buckets: buckets,
		objects: objects,
		storage: storage,
	}
}

func (s *ObjectService) UploadObject(
	ctx context.Context,
	bucketName string,
	objectKey string,
	contentType string,
	size int64,
	body io.Reader,
) (*models.Object, error) {

	bucket, err := s.buckets.FindByName(ctx, bucketName)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, ErrBucketNotFound
	}

	id := uuid.New()

	shard := id.String()[:2]
	storagePath := filepath.Join(shard, id.String())

	hasher := sha256.New()

	reader := io.TeeReader(body, hasher)

	err = s.storage.Save(storagePath, reader)
	if err != nil {
		return nil, err
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	existingObject, _ := s.objects.FindByKey(
	ctx,
	bucket.ID,
	objectKey,
)

object := &models.Object{
	ID:          id,
	BucketID:    bucket.ID,
	ObjectKey:   objectKey,
	StoragePath: storagePath,
	ContentType: contentType,
	SizeBytes:   size,
	Checksum:    checksum,
}

	err = s.objects.Upsert(ctx, object)
	if err != nil {

		cleanupErr := s.storage.Delete(storagePath)

		if cleanupErr != nil {
			config.AppLogger.Printf(
				"cleanup failed after db error: %v",
				cleanupErr,
			)
		}

		return nil, err
	}

	if existingObject != nil {

		err = s.storage.Delete(
			existingObject.StoragePath,
		)

		if err != nil {
			config.AppLogger.Printf(
				"failed removing old file: %v",
				err,
			)
		}
	}

	return object, nil
}

func (s *ObjectService) GetObject(
	ctx context.Context,
	bucketName string,
	objectKey string,
) (*models.Object, error) {

	bucket, err := s.buckets.FindByName(ctx, bucketName)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, ErrBucketNotFound
	}

	object, err := s.objects.FindByKey(
		ctx,
		bucket.ID,
		objectKey,
	)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (s *ObjectService) OpenObjectFile(
	storagePath string,
) (io.ReadCloser, error) {

	return s.storage.Open(storagePath)
}

func (s *ObjectService) DeleteObject(
	ctx context.Context,
	bucketName string,
	objectKey string,
) error {

	bucket, err := s.buckets.FindByName(
		ctx,
		bucketName,
	)
	if err != nil {
		return err
	}

	if bucket == nil {
		return ErrBucketNotFound
	}

	object, err := s.objects.FindByKey(
		ctx,
		bucket.ID,
		objectKey,
	)
	if err != nil {
		return err
	}

	err = s.storage.Delete(
		object.StoragePath,
	)
	if err != nil {
		return err
	}

	err = s.objects.Delete(
		ctx,
		object.ID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *ObjectService) ListObjects(
	ctx context.Context,
	bucketName string,
) ([]models.Object, error) {

	bucket, err := s.buckets.FindByName(ctx, bucketName)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, ErrBucketNotFound
	}

	return s.objects.ListByBucket(ctx, bucket.ID)
}