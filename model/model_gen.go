// Code generated by bit. DO NOT EDIT.

package model

import (
	"database/sql/driver"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"time"
)

type Array []interface{}

func (x *Array) Scan(input interface{}) error {
	return jsoniter.Unmarshal(input.([]byte), x)
}

func (x Array) Value() (driver.Value, error) {
	return jsoniter.Marshal(x)
}

type Object map[string]interface{}

func (x *Object) Scan(input interface{}) error {
	return jsoniter.Unmarshal(input.([]byte), x)
}

func (x Object) Value() (driver.Value, error) {
	return jsoniter.Marshal(x)
}

func True() *bool {
	value := true
	return &value
}

func False() *bool {
	return new(bool)
}

type Role struct {
	ID          int64
	Status      *bool
	CreateTime  time.Time
	UpdateTime  time.Time
	Key         string
	Name        string
	Routers     ref
	Description string
	Permissions ref
}

type Admin struct {
	ID          int64
	Status      *bool
	CreateTime  time.Time
	UpdateTime  time.Time
	Phone       string
	Avatar      Array
	Password    string
	Permissions ref
	Routers     ref
	Username    string
	Name        string
	Uuid        uuid.UUID
	Email       string
	Roles       ref
}
