package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVCanName(t *testing.T) {
	assert.Equal(t, "vcan-0", vcanName("0"))
	assert.Equal(t, "vcan-S10..EXT-1", vcanName("S101-IOU04-USB-EXT-1-can"))
	assert.Equal(t, "vcan-MIO04-1", vcanName("MIO04-1-can"))
}
