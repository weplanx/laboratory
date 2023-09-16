package model_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/weplanx/server/model"
	"testing"
)

func TestSetSchedules(t *testing.T) {
	ctx := context.TODO()
	err := model.SetSchedules(ctx, x.Db)
	assert.NoError(t, err)
}
