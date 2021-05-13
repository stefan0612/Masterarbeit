package test

import (
	"context"
	"smb-csi/driver/mock"
)

var ctx = context.Background()
var d, _ = mock.NewMockDriver("test")
