package helpers

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCoalesce(t *testing.T) {
	result := Coalesce("", "b", "c")
	assert.Equal(t, result, "b")
}

func TestDetectComposeFile(t *testing.T) {
	cf := DetectComposeFile()
	assert.Equal(t, cf, "docker-compose.yml")

	os.Setenv("COMPOSE_FILE", "foo")
	cf = DetectComposeFile()
	assert.Equal(t, cf, "foo")
}

func TestExists(t *testing.T) {
	result := Exists("helpers_test.go")
	assert.Equal(t, result, true)

	result = Exists("nonexistent")
	assert.Equal(t, result, false)
}

func TestHumanizeTime(t *testing.T) {
	now := time.Now()
	result := HumanizeTime(now)
	assert.Equal(t, result, "now")

	zero := time.Time{}
	result = HumanizeTime(zero)
	assert.Equal(t, result, "")
}

func TestIn(t *testing.T) {
	words := []string{"I", "am", "a", "traveler", "of", "both", "time", "and", "space,", "to", "be", "where", "I", "have", "been"}
	assert.Equal(t, In("traveler", words), true)
	assert.Equal(t, In("kashmir", words), false)
}
