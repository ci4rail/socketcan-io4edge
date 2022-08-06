package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVCanName(t *testing.T) {
	assert.Equal(t, "vcan0", vcanName("0"))
	assert.Equal(t, "vcanS101xxEXT-1", vcanName("S101-IOU04-USB-EXT-1-can"))
	assert.Equal(t, "vcanMIO04-1", vcanName("MIO04-1-can"))
	assert.Equal(t, "vcan12345678901", vcanName("12345678901"))
	assert.Equal(t, "vcan1234xx89012", vcanName("123456789012"))
}
