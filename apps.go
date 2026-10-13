// 本文件实现了 Steam Apps handler，对应 SteamKit2 的 Steam/Handlers/SteamApps。
// 基于对 SteamKit2 源码的理解：构造 CMsgClientRequestFreeLicense protobuf，
// 设置 SourceJobID，发送后通过 JobID 匹配 EMsgClientRequestFreeLicenseResponse 响应。
package steam

import (
	"errors"
	"sync"
	"time"

	"github.com/Philipp15b/go-steam/v3/protocol"
	"github.com/Philipp15b/go-steam/v3/protocol/protobuf"
	"github.com/Philipp15b/go-steam/v3/protocol/steamlang"
)

// FreeLicenseResult 是一次免费许可请求的结果（对应 SteamKit2 的 FreeLicenseCallback）。
type FreeLicenseResult struct {
	Result          steamlang.EResult
	GrantedApps     []uint32
	GrantedPackages []uint32
}

// Apps 对应 SteamKit2 的 SteamApps handler，处理应用/许可相关的 CM 请求。
type Apps struct {
	client *Client
	mutex  sync.Mutex
	jobs   map[protocol.JobId]chan *FreeLicenseResult
}

// newApps 创建 Apps handler 并注册到客户端。
func newApps(client *Client) *Apps {
	apps := &Apps{client: client, jobs: make(map[protocol.JobId]chan *FreeLicenseResult)}
	client.RegisterPacketHandler(apps)
	return apps
}

// HandlePacket 实现 PacketHandler 接口：处理免费许可响应。
func (a *Apps) HandlePacket(packet *protocol.Packet) {
	if packet.EMsg != steamlang.EMsg_ClientRequestFreeLicenseResponse {
		return
	}

	body := new(protobuf.CMsgClientRequestFreeLicenseResponse)
	packet.ReadProtoMsg(body)

	a.mutex.Lock()
	ch := a.jobs[packet.TargetJobId]
	delete(a.jobs, packet.TargetJobId)
	a.mutex.Unlock()

	if ch == nil {
		return
	}

	ch <- &FreeLicenseResult{
		Result:           steamlang.EResult(body.GetEresult()),
		GrantedApps:      body.GetGrantedAppids(),
		GrantedPackages: body.GetGrantedPackageids(),
	}
}

// RequestFreeLicense 请求单个 appID 的免费许可，返回激活结果。
// 对应 SteamKit2 的 SteamApps.RequestFreeLicense(app)。
func (a *Apps) RequestFreeLicense(appID uint32) (*FreeLicenseResult, error) {
	return a.RequestFreeLicenseMulti([]uint32{appID})
}

// RequestFreeLicenseMulti 请求多个 appID 的免费许可，返回激活结果。
func (a *Apps) RequestFreeLicenseMulti(appIDs []uint32) (*FreeLicenseResult, error) {
	if len(appIDs) == 0 {
		return nil, errors.New("appIDs 不能为空")
	}

	jobID := a.client.GetNextJobId()
	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientRequestFreeLicense, &protobuf.CMsgClientRequestFreeLicense{
		Appids: appIDs,
	})
	msg.SetSourceJobId(jobID)

	ch := make(chan *FreeLicenseResult, 1)
	a.mutex.Lock()
	a.jobs[jobID] = ch
	a.mutex.Unlock()

	a.client.Write(msg)

	select {
	case result := <-ch:
		return result, nil
	case <-time.After(30 * time.Second):
		a.mutex.Lock()
		delete(a.jobs, jobID)
		a.mutex.Unlock()
		return nil, errors.New("等待免费许可响应超时")
	}
}