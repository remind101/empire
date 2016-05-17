mocks: mocks/kinesis.go

clean:
	rm -rf build/*

build/kinesumer:
	go build -o build/kinesumer ./cmd/kinesumer

mocks/kinesis.go:
	mockery \
		-dir=./Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis/kinesisiface \
		-name=KinesisAPI -print=true \
	| sed -e s/KinesisAPI/Kinesis/g > mocks/kinesis.go 
