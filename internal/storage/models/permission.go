package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	PermissionBucketCreate = "bucket:create"
	PermissionBucketList   = "bucket:list"
	PermissionBucketDelete = "bucket:delete"

	PermissionObjectUpload   = "object:upload"
	PermissionObjectDownload = "object:download"
	PermissionObjectDelete   = "object:delete"
	PermissionObjectList     = "object:list"
	PermissionObjectSign     = "object:sign"

	PermissionAPIKeyCreate = "apikey:create"
	PermissionAPIKeyList   = "apikey:list"
	PermissionAPIKeyDelete = "apikey:delete"

	PermissionUserCreate          = "user:create"
	PermissionUserList            = "user:list"
	PermissionUserGrantPermission = "user:grant-permission"
)

var AllPermissionNames = []string{
	PermissionBucketCreate,
	PermissionBucketList,
	PermissionBucketDelete,
	PermissionObjectUpload,
	PermissionObjectDownload,
	PermissionObjectDelete,
	PermissionObjectList,
	PermissionAPIKeyCreate,
	PermissionAPIKeyList,
	PermissionAPIKeyDelete,
	PermissionUserCreate,
	PermissionUserList,
	PermissionUserGrantPermission,
	PermissionObjectSign,
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
