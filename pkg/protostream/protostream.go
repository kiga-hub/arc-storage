package protostream

import (
	pb "github.com/kiga-hub/arc/protobuf/pb"
)

// ProtoStream -
type ProtoStream struct {
	Key    []byte
	Value  []byte
	IsStop bool
}

// FrameData -
type FrameData struct {
	Grpcmessage chan ProtoStream
}

// FrameDataCallback -
func (t *FrameData) FrameDataCallback(request pb.FrameData_FrameDataCallbackServer) (err error) {
	ctx := request.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tem, err := request.Recv()
		if err != nil {
			return request.SendAndClose(&pb.FrameDataResponse{Successed: false})
		}
		message := &ProtoStream{
			Key:    tem.Key,
			Value:  tem.Value,
			IsStop: false,
		}

		t.Grpcmessage <- *message
	}
}
