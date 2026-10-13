package steam

import (
	"bytes"
	"testing"

	"github.com/Philipp15b/go-steam/v3/protocol"
	"github.com/Philipp15b/go-steam/v3/protocol/protobuf"
	"github.com/Philipp15b/go-steam/v3/protocol/steamlang"
	"google.golang.org/protobuf/proto"
)

func TestAppsRegistered(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	if client.Apps == nil {
		t.Fatal("Apps handler 未注册")
	}
	if _, err := client.Apps.RequestFreeLicenseMulti(nil); err == nil {
		t.Error("空 appIDs 应返回错误")
	}
}

func TestAppsHandlePacket(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	jobID := client.GetNextJobId()
	ch := make(chan *FreeLicenseResult, 1)
	client.Apps.mutex.Lock()
	client.Apps.jobs[jobID] = ch
	client.Apps.mutex.Unlock()

	// 构造 EMsgClientRequestFreeLicenseResponse 响应包（header + protobuf body）
	body := &protobuf.CMsgClientRequestFreeLicenseResponse{
		Eresult:           proto.Uint32(uint32(steamlang.EResult_OK)),
		GrantedAppids:     []uint32{123},
		GrantedPackageids: []uint32{456},
	}
	header := steamlang.NewMsgHdrProtoBuf()
	header.Msg = steamlang.EMsg_ClientRequestFreeLicenseResponse
	header.Proto.JobidTarget = proto.Uint64(uint64(jobID))
	var headerBuf bytes.Buffer
	if err := header.Serialize(&headerBuf); err != nil {
		t.Fatalf("序列化 header 失败: %v", err)
	}
	bodyData, err := proto.Marshal(body)
	if err != nil {
		t.Fatalf("序列化 body 失败: %v", err)
	}

	packet := &protocol.Packet{
		EMsg:        steamlang.EMsg_ClientRequestFreeLicenseResponse,
		IsProto:     true,
		TargetJobId: jobID,
		Data:        append(headerBuf.Bytes(), bodyData...),
	}

	client.Apps.HandlePacket(packet)

	select {
	case result := <-ch:
		if result == nil {
			t.Fatal("结果为 nil")
		}
		if result.Result != steamlang.EResult_OK {
			t.Errorf("Result = %v, 期望 OK", result.Result)
		}
		if len(result.GrantedApps) != 1 || result.GrantedApps[0] != 123 {
			t.Errorf("GrantedApps = %v, 期望 [123]", result.GrantedApps)
		}
		if len(result.GrantedPackages) != 1 || result.GrantedPackages[0] != 456 {
			t.Errorf("GrantedPackages = %v, 期望 [456]", result.GrantedPackages)
		}
	default:
		t.Error("未收到结果")
	}
}