package pbjs

import (
	"strings"

	"github.com/nats-io/nats.go"
)

const (
	HdrNatsMsgID     = nats.MsgIdHdr
	HdrNatsTimestamp = "Nats-Time-Stamp"
	HdrMsgType       = "Msg-Type"
	HdrContentType   = "Content-Type"
	HdrCorrelationID = "Correlation-Id"
	HdrSubject       = "Subject"

	hdrExpectPrefix  = "Nats-Expected-"
	mimeTypeProtobuf = "application/protobuf"
)

func removeExpectHeaders(h nats.Header) {
	for k := range h {
		if strings.HasPrefix(k, hdrExpectPrefix) {
			delete(h, k)
		}
	}
}
