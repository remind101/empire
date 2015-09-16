package kinesumeriface

import (
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
)

type Kinesis kinesisiface.KinesisAPI
