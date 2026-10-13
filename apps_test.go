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

func TestAppsHandleFreeLicensePacket(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	jobID := client.GetNextJobId()
	ch := make(chan interface{}, 1)
	client.Apps.mutex.Lock()
	client.Apps.jobs[jobID] = ch
	client.Apps.mutex.Unlock()

	// 构造 EMsgClientRequestFreeLicenseResponse 响应包（header + protobuf body）
	body := &protobuf.CMsgClientRequestFreeLicenseResponse{
		Eresult:           proto.Uint32(uint32(steamlang.EResult_OK)),
		GrantedAppids:       []uint32{123},
		GrantedPackageids:   []uint32{456},
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
	case raw := <-ch:
		result, ok := raw.(*FreeLicenseResult)
		if !ok {
			t.Fatalf("类型断言失败: %T", raw)
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

func TestAppsHandleCheckAppBetaPasswordPacket(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	jobID := client.GetNextJobId()
	ch := make(chan interface{}, 1)
	client.Apps.mutex.Lock()
	client.Apps.jobs[jobID] = ch
	client.Apps.mutex.Unlock()

	// 构造 EMsgClientCheckAppBetaPasswordResponse 响应包
	body := &protobuf.CMsgClientCheckAppBetaPasswordResponse{
		Eresult: proto.Int32(int32(steamlang.EResult_OK)),
		Betapasswords: []*protobuf.CMsgClientCheckAppBetaPasswordResponse_BetaPassword{
			{Betaname: proto.String("beta"), Betapassword: proto.String("deadbeef")},
		},
	}
	header := steamlang.NewMsgHdrProtoBuf()
	header.Msg = steamlang.EMsg_ClientCheckAppBetaPasswordResponse
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
		EMsg:        steamlang.EMsg_ClientCheckAppBetaPasswordResponse,
		IsProto:     true,
		TargetJobId: jobID,
		Data:        append(headerBuf.Bytes(), bodyData...),
	}

	client.Apps.HandlePacket(packet)

	select {
	case raw := <-ch:
		result, ok := raw.(*CheckAppBetaPasswordResult)
		if !ok {
			t.Fatalf("类型断言失败: %T", raw)
		}
		if result.Result != steamlang.EResult_OK {
			t.Errorf("Result = %v, 期望 OK", result.Result)
		}
		if len(result.BetaPasswords) != 1 {
			t.Fatalf("BetaPasswords 长度 = %d, 期望 1", len(result.BetaPasswords))
		}
		if result.BetaPasswords[0].Name != "beta" {
			t.Errorf("BetaPasswords[0].Name = %q, 期望 %q", result.BetaPasswords[0].Name, "beta")
		}
		if string(result.BetaPasswords[0].Password) != "\xde\xad\xbe\xef" {
			t.Errorf("BetaPasswords[0].Password = %x, 期望 deadbeef", result.BetaPasswords[0].Password)
		}
	default:
		t.Error("未收到结果")
	}
}

func TestAppsHandleGetAppOwnershipTicketPacket(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	jobID := client.GetNextJobId()
	ch := make(chan interface{}, 1)
	client.Apps.mutex.Lock()
	client.Apps.jobs[jobID] = ch
	client.Apps.mutex.Unlock()

	body := &protobuf.CMsgClientGetAppOwnershipTicketResponse{
		Eresult: proto.Uint32(uint32(steamlang.EResult_OK)),
		AppId:   proto.Uint32(730),
		Ticket:  []byte("ticket-data"),
	}
	header := steamlang.NewMsgHdrProtoBuf()
	header.Msg = steamlang.EMsg_ClientGetAppOwnershipTicketResponse
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
		EMsg:        steamlang.EMsg_ClientGetAppOwnershipTicketResponse,
		IsProto:     true,
		TargetJobId: jobID,
		Data:        append(headerBuf.Bytes(), bodyData...),
	}

	client.Apps.HandlePacket(packet)

	select {
	case raw := <-ch:
		result, ok := raw.(*AppOwnershipTicketResult)
		if !ok {
			t.Fatalf("类型断言失败: %T", raw)
		}
		if result.Result != steamlang.EResult_OK {
			t.Errorf("Result = %v, 期望 OK", result.Result)
		}
		if result.AppID != 730 {
			t.Errorf("AppID = %d, 期望 730", result.AppID)
		}
		if string(result.Ticket) != "ticket-data" {
			t.Errorf("Ticket = %q, 期望 %q", result.Ticket, "ticket-data")
		}
	default:
		t.Error("未收到结果")
	}
}

func TestAppsHandlePICSGetAccessTokensPacket(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	jobID := client.GetNextJobId()
	ch := make(chan interface{}, 1)
	client.Apps.mutex.Lock()
	client.Apps.jobs[jobID] = ch
	client.Apps.mutex.Unlock()

	body := &protobuf.CMsgClientPICSAccessTokenResponse{
		AppAccessTokens: []*protobuf.CMsgClientPICSAccessTokenResponse_AppToken{
			{Appid: proto.Uint32(730), AccessToken: proto.Uint64(111)},
		},
		AppDeniedTokens: []uint32{999},
		PackageAccessTokens: []*protobuf.CMsgClientPICSAccessTokenResponse_PackageToken{
			{Packageid: proto.Uint32(456), AccessToken: proto.Uint64(222)},
		},
	}
	header := steamlang.NewMsgHdrProtoBuf()
	header.Msg = steamlang.EMsg_ClientPICSAccessTokenResponse
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
		EMsg:        steamlang.EMsg_ClientPICSAccessTokenResponse,
		IsProto:     true,
		TargetJobId: jobID,
		Data:        append(headerBuf.Bytes(), bodyData...),
	}

	client.Apps.HandlePacket(packet)

	select {
	case raw := <-ch:
		result, ok := raw.(*PICSTokensResult)
		if !ok {
			t.Fatalf("类型断言失败: %T", raw)
		}
		if len(result.AppAccessTokens) != 1 || result.AppAccessTokens[0].AppID != 730 || result.AppAccessTokens[0].AccessToken != 111 {
			t.Errorf("AppAccessTokens = %v", result.AppAccessTokens)
		}
		if len(result.AppDeniedTokens) != 1 || result.AppDeniedTokens[0] != 999 {
			t.Errorf("AppDeniedTokens = %v", result.AppDeniedTokens)
		}
		if len(result.PackageAccessTokens) != 1 || result.PackageAccessTokens[0].PackageID != 456 || result.PackageAccessTokens[0].AccessToken != 222 {
			t.Errorf("PackageAccessTokens = %v", result.PackageAccessTokens)
		}
	default:
		t.Error("未收到结果")
	}
}

func TestAppsHandleGetDepotDecryptionKeyPacket(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	jobID := client.GetNextJobId()
	ch := make(chan interface{}, 1)
	client.Apps.mutex.Lock()
	client.Apps.jobs[jobID] = ch
	client.Apps.mutex.Unlock()

	body := &protobuf.CMsgClientGetDepotDecryptionKeyResponse{
		Eresult:            proto.Int32(int32(steamlang.EResult_OK)),
		DepotId:            proto.Uint32(2347770),
		DepotEncryptionKey: []byte("depot-key"),
	}
	header := steamlang.NewMsgHdrProtoBuf()
	header.Msg = steamlang.EMsg_ClientGetDepotDecryptionKeyResponse
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
		EMsg:        steamlang.EMsg_ClientGetDepotDecryptionKeyResponse,
		IsProto:     true,
		TargetJobId: jobID,
		Data:        append(headerBuf.Bytes(), bodyData...),
	}

	client.Apps.HandlePacket(packet)

	select {
	case raw := <-ch:
		result, ok := raw.(*DepotKeyResult)
		if !ok {
			t.Fatalf("类型断言失败: %T", raw)
		}
		if result.Result != steamlang.EResult_OK {
			t.Errorf("Result = %v, 期望 OK", result.Result)
		}
		if result.DepotID != 2347770 {
			t.Errorf("DepotID = %d, 期望 2347770", result.DepotID)
		}
		if string(result.EncryptionKey) != "depot-key" {
			t.Errorf("EncryptionKey = %q, 期望 %q", result.EncryptionKey, "depot-key")
		}
	default:
		t.Error("未收到结果")
	}
}

func TestAppsHandlePICSChangesSincePacket(t *testing.T) {
	client := NewClient()
	defer client.Disconnect()

	jobID := client.GetNextJobId()
	ch := make(chan interface{}, 1)
	client.Apps.mutex.Lock()
	client.Apps.jobs[jobID] = ch
	client.Apps.mutex.Unlock()

	body := &protobuf.CMsgClientPICSChangesSinceResponse{
		CurrentChangeNumber: proto.Uint32(100),
		SinceChangeNumber:  proto.Uint32(90),
		ForceFullUpdate:    proto.Bool(false),
		AppChanges: []*protobuf.CMsgClientPICSChangesSinceResponse_AppChange{
			{Appid: proto.Uint32(730), ChangeNumber: proto.Uint32(95), NeedsToken: proto.Bool(true)},
		},
		PackageChanges: []*protobuf.CMsgClientPICSChangesSinceResponse_PackageChange{
			{Packageid: proto.Uint32(456), ChangeNumber: proto.Uint32(92)},
		},
	}
	header := steamlang.NewMsgHdrProtoBuf()
	header.Msg = steamlang.EMsg_ClientPICSChangesSinceResponse
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
		EMsg:        steamlang.EMsg_ClientPICSChangesSinceResponse,
		IsProto:     true,
		TargetJobId: jobID,
		Data:        append(headerBuf.Bytes(), bodyData...),
	}

	client.Apps.HandlePacket(packet)

	select {
	case raw := <-ch:
		result, ok := raw.(*PICSChangesResult)
		if !ok {
			t.Fatalf("类型断言失败: %T", raw)
		}
		if result.CurrentChangeNumber != 100 || result.SinceChangeNumber != 90 {
			t.Errorf("变更号 = %d/%d", result.CurrentChangeNumber, result.SinceChangeNumber)
		}
		if len(result.AppChanges) != 1 || result.AppChanges[0].AppID != 730 || result.AppChanges[0].ChangeNumber != 95 || !result.AppChanges[0].NeedsToken {
			t.Errorf("AppChanges = %v", result.AppChanges)
		}
		if len(result.PackageChanges) != 1 || result.PackageChanges[0].PackageID != 456 || result.PackageChanges[0].ChangeNumber != 92 {
			t.Errorf("PackageChanges = %v", result.PackageChanges)
		}
	default:
		t.Error("未收到结果")
	}
}