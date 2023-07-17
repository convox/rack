package jwt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/jwt"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/assert"
)

func TestJwtReadToken(t *testing.T) {
	jm := jwt.NewJwtManager("TEST")

	tk, err := jm.ReadToken(time.Hour)
	assert.NoError(t, err, "no error")

	data, err := jm.Verify(tk)
	assert.NoError(t, err)
	assert.Equal(t, data.Role, structs.ConvoxRoleRead)
}

func TestJwtReadTokenExpired(t *testing.T) {
	jm := jwt.NewJwtManager("TEST")

	tk, err := jm.ReadToken(time.Hour * -1)
	assert.NoError(t, err, "no error")

	data, err := jm.Verify(tk)
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestJwtReadTokenInvalid(t *testing.T) {
	jm := jwt.NewJwtManager("TEST")

	tk, err := jm.ReadToken(time.Hour * -1)
	assert.NoError(t, err, "no error")

	data, err := jm.Verify(tk[:len(tk)-1])
	fmt.Println(err)
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestJwtWriteToken(t *testing.T) {
	jm := jwt.NewJwtManager("TEST")

	tk, err := jm.WriteToken(time.Hour)
	assert.NoError(t, err, "no error")

	data, err := jm.Verify(tk)
	assert.NoError(t, err)
	assert.Equal(t, data.Role, structs.ConvoxRoleReadWrite)
}
