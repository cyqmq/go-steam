// 本文件实现了 Steam Apps handler，对应 SteamKit2 的 Steam/Handlers/SteamApps。
// 基于对 SteamKit2 源码的理解：构造对应 EMsg 的 protobuf 消息，
// 设置 SourceJobID，发送后通过 JobID 匹配响应消息。
package steam

import (
	"errors"
	"sync"
	"time"

	"github.com/Philipp15b/go-steam/v3/protocol"
	"github.com/Philipp15b/go-steam/v3/protocol/protobuf"
	"github.com/Philipp15b/go-steam/v3/protocol/steamlang"
	"google.golang.org/protobuf/proto"
)

// FreeLicenseResult 是一次免费许可请求的结果（对应 SteamKit2 的 FreeLicenseCallback）。
type FreeLicenseResult struct {
	Result          steamlang.EResult
	GrantedApps     []uint32
	GrantedPackages []uint32
}

// BetaPassword 是一个 beta 分支的名称与加密密钥（对应 SteamKit2 的 BetaPasswords 字典项）。
type BetaPassword struct {
	Name     string
	Password []byte
}

// CheckAppBetaPasswordResult 是一次 beta 密码检查的结果（对应 SteamKit2 的 CheckAppBetaPasswordCallback）。
type CheckAppBetaPasswordResult struct {
	Result        steamlang.EResult
	BetaPasswords []BetaPassword
}

// Apps 对应 SteamKit2 的 SteamApps handler，处理应用/许可相关的 CM 请求。
type Apps struct {
	client *Client
	mutex  sync.Mutex
	jobs   map[protocol.JobId]chan interface{}
}

// newApps 创建 Apps handler 并注册到客户端。
func newApps(client *Client) *Apps {
	apps := &Apps{client: client, jobs: make(map[protocol.JobId]chan interface{})}
	client.RegisterPacketHandler(apps)
	return apps
}

// HandlePacket 实现 PacketHandler 接口：处理 Apps 相关的响应消息。
func (a *Apps) HandlePacket(packet *protocol.Packet) {
	var result interface{}

	switch packet.EMsg {
	case steamlang.EMsg_ClientRequestFreeLicenseResponse:
		body := new(protobuf.CMsgClientRequestFreeLicenseResponse)
		packet.ReadProtoMsg(body)
		result = &FreeLicenseResult{
			Result:           steamlang.EResult(body.GetEresult()),
			GrantedApps:      body.GetGrantedAppids(),
			GrantedPackages: body.GetGrantedPackageids(),
		}
	case steamlang.EMsg_ClientCheckAppBetaPasswordResponse:
		body := new(protobuf.CMsgClientCheckAppBetaPasswordResponse)
		packet.ReadProtoMsg(body)
		betaPasswords := make([]BetaPassword, 0, len(body.GetBetapasswords()))
		for _, bp := range body.GetBetapasswords() {
			betaPasswords = append(betaPasswords, BetaPassword{
				Name:     bp.GetBetaname(),
				Password: []byte(bp.GetBetapassword()),
			})
		}
		result = &CheckAppBetaPasswordResult{
			Result:        steamlang.EResult(body.GetEresult()),
			BetaPasswords: betaPasswords,
		}
	default:
		return
	}

	a.mutex.Lock()
	ch := a.jobs[packet.TargetJobId]
	delete(a.jobs, packet.TargetJobId)
	a.mutex.Unlock()

	if ch != nil {
		ch <- result
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

	ch := make(chan interface{}, 1)
	a.mutex.Lock()
	a.jobs[jobID] = ch
	a.mutex.Unlock()

	a.client.Write(msg)

	select {
	case raw := <-ch:
		return raw.(*FreeLicenseResult), nil
	case <-time.After(30 * time.Second):
		a.mutex.Lock()
		delete(a.jobs, jobID)
		a.mutex.Unlock()
		return nil, errors.New("等待免费许可响应超时")
	}
}

// CheckAppBetaPassword 提交指定 appID 的 beta 密码，返回可用 beta 分支及其加密密钥。
// 对应 SteamKit2 的 SteamApps.CheckAppBetaPassword(app, password)。
func (a *Apps) CheckAppBetaPassword(appID uint32, password string) (*CheckAppBetaPasswordResult, error) {
	jobID := a.client.GetNextJobId()
	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientCheckAppBetaPassword, &protobuf.CMsgClientCheckAppBetaPassword{
		AppId:        proto.Uint32(appID),
		Betapassword: proto.String(password),
	})
	msg.SetSourceJobId(jobID)

	ch := make(chan interface{}, 1)
	a.mutex.Lock()
	a.jobs[jobID] = ch
	a.mutex.Unlock()

	a.client.Write(msg)

	select {
	case raw := <-ch:
		return raw.(*CheckAppBetaPasswordResult), nil
	case <-time.After(30 * time.Second):
		a.mutex.Lock()
		delete(a.jobs, jobID)
		a.mutex.Unlock()
		return nil, errors.New("等待 beta 密码检查响应超时")
	}
}