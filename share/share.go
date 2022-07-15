package share

import (
	"github.com/hezhis/net_rpc/codec"
	"github.com/hezhis/net_rpc/protocol"
)

// Codecs are codecs supported by rpcx. You can add customized codecs in Codecs.
var Codecs = map[protocol.SerializeType]codec.Codec{
	protocol.SerializeNone: nil,
	protocol.ProtoBuffer:   &codec.PBCodec{},
}
