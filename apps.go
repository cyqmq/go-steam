// 本文件实现了 Steam Apps handler，对应 SteamKit2 的 Steam/Handlers/SteamApps。
// 基于对 SteamKit2 源码的理解：构造对应 EMsg 的 protobuf 消息，
// 设置 SourceJobID，发送后通过 JobID 匹配响应消息。
package steam

import (
	"encoding/hex"
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

// AppOwnershipTicketResult 是一次所有权票据请求的结果（对应 SteamKit2 的 AppOwnershipTicketCallback）。
type AppOwnershipTicketResult struct {
	Result steamlang.EResult
	AppID  uint32
	Ticket []byte
}

// PICSAppToken 是一个 app 的 PICS 访问令牌。
type PICSAppToken struct {
	AppID       uint32
	AccessToken uint64
}

// PICSPackageToken 是一个 package 的 PICS 访问令牌。
type PICSPackageToken struct {
	PackageID   uint32
	AccessToken uint64
}

// PICSTokensResult 是 PICS 访问令牌请求的结果（对应 SteamKit2 的 PICSTokensCallback）。
type PICSTokensResult struct {
	AppAccessTokens      []PICSAppToken
	AppDeniedTokens      []uint32
	PackageAccessTokens []PICSPackageToken
	PackageDeniedTokens  []uint32
}

// DepotKeyResult 是一次 depot 解密密钥请求的结果（对应 SteamKit2 的 DepotKeyCallback）。
type DepotKeyResult struct {
	Result        steamlang.EResult
	DepotID       uint32
	EncryptionKey []byte
}

// PICSAppChange 是一个 app 的 PICS 变更项。
type PICSAppChange struct {
	AppID        uint32
	ChangeNumber uint32
	NeedsToken   bool
}

// PICSPackageChange 是一个 package 的 PICS 变更项。
type PICSPackageChange struct {
	PackageID    uint32
	ChangeNumber uint32
	NeedsToken   bool
}

// PICSChangesResult 是 PICS 变更查询的结果（对应 SteamKit2 的 PICSChangesCallback）。
type PICSChangesResult struct {
	CurrentChangeNumber uint32
	SinceChangeNumber  uint32
	ForceFullUpdate    bool
	AppChanges          []PICSAppChange
	PackageChanges      []PICSPackageChange
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
			// SteamKit2: BetaPasswords[betaname] = Utils.DecodeHexString(betapassword)，betapassword 是十六进制字符串
			password, err := hex.DecodeString(bp.GetBetapassword())
			if err != nil {
				password = []byte(bp.GetBetapassword())
			}
			betaPasswords = append(betaPasswords, BetaPassword{
				Name:     bp.GetBetaname(),
				Password: password,
			})
		}
		result = &CheckAppBetaPasswordResult{
			Result:        steamlang.EResult(body.GetEresult()),
			BetaPasswords: betaPasswords,
		}
	case steamlang.EMsg_ClientGetAppOwnershipTicketResponse:
		body := new(protobuf.CMsgClientGetAppOwnershipTicketResponse)
		packet.ReadProtoMsg(body)
		result = &AppOwnershipTicketResult{
			Result: steamlang.EResult(body.GetEresult()),
			AppID:   body.GetAppId(),
			Ticket:  body.GetTicket(),
		}
	case steamlang.EMsg_ClientPICSAccessTokenResponse:
		body := new(protobuf.CMsgClientPICSAccessTokenResponse)
		packet.ReadProtoMsg(body)
		appTokens := make([]PICSAppToken, 0, len(body.GetAppAccessTokens()))
		for _, t := range body.GetAppAccessTokens() {
			appTokens = append(appTokens, PICSAppToken{AppID: t.GetAppid(), AccessToken: t.GetAccessToken()})
		}
		packageTokens := make([]PICSPackageToken, 0, len(body.GetPackageAccessTokens()))
		for _, t := range body.GetPackageAccessTokens() {
			packageTokens = append(packageTokens, PICSPackageToken{PackageID: t.GetPackageid(), AccessToken: t.GetAccessToken()})
		}
		result = &PICSTokensResult{
			AppAccessTokens:      appTokens,
			AppDeniedTokens:      body.GetAppDeniedTokens(),
			PackageAccessTokens: packageTokens,
			PackageDeniedTokens: body.GetPackageDeniedTokens(),
		}
	case steamlang.EMsg_ClientGetDepotDecryptionKeyResponse:
		body := new(protobuf.CMsgClientGetDepotDecryptionKeyResponse)
		packet.ReadProtoMsg(body)
		result = &DepotKeyResult{
			Result:        steamlang.EResult(body.GetEresult()),
			DepotID:       body.GetDepotId(),
			EncryptionKey: body.GetDepotEncryptionKey(),
		}
	case steamlang.EMsg_ClientPICSChangesSinceResponse:
		body := new(protobuf.CMsgClientPICSChangesSinceResponse)
		packet.ReadProtoMsg(body)
		appChanges := make([]PICSAppChange, 0, len(body.GetAppChanges()))
		for _, ch := range body.GetAppChanges() {
			appChanges = append(appChanges, PICSAppChange{
				AppID:        ch.GetAppid(),
				ChangeNumber: ch.GetChangeNumber(),
				NeedsToken:   ch.GetNeedsToken(),
			})
		}
		packageChanges := make([]PICSPackageChange, 0, len(body.GetPackageChanges()))
		for _, ch := range body.GetPackageChanges() {
			packageChanges = append(packageChanges, PICSPackageChange{
				PackageID:    ch.GetPackageid(),
				ChangeNumber: ch.GetChangeNumber(),
				NeedsToken:   ch.GetNeedsToken(),
			})
		}
		result = &PICSChangesResult{
			CurrentChangeNumber: body.GetCurrentChangeNumber(),
			SinceChangeNumber:  body.GetSinceChangeNumber(),
			ForceFullUpdate:    body.GetForceFullUpdate(),
			AppChanges:          appChanges,
			PackageChanges:      packageChanges,
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

	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientRequestFreeLicense, &protobuf.CMsgClientRequestFreeLicense{
		Appids: appIDs,
	})
	jobID := a.client.GetNextJobId()
	msg.SetSourceJobId(jobID)

	raw, err := a.sendJob(msg, jobID)
	if err != nil {
		return nil, err
	}
	return raw.(*FreeLicenseResult), nil
}

// CheckAppBetaPassword 提交指定 appID 的 beta 密码，返回可用 beta 分支及其加密密钥。
// 对应 SteamKit2 的 SteamApps.CheckAppBetaPassword(app, password)。
func (a *Apps) CheckAppBetaPassword(appID uint32, password string) (*CheckAppBetaPasswordResult, error) {
	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientCheckAppBetaPassword, &protobuf.CMsgClientCheckAppBetaPassword{
		AppId:        proto.Uint32(appID),
		Betapassword: proto.String(password),
	})
	jobID := a.client.GetNextJobId()
	msg.SetSourceJobId(jobID)

	raw, err := a.sendJob(msg, jobID)
	if err != nil {
		return nil, err
	}
	return raw.(*CheckAppBetaPasswordResult), nil
}

// GetAppOwnershipTicket 请求指定 appID 的所有权票据。
// 对应 SteamKit2 的 SteamApps.GetAppOwnershipTicket(appid)。
func (a *Apps) GetAppOwnershipTicket(appID uint32) (*AppOwnershipTicketResult, error) {
	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientGetAppOwnershipTicket, &protobuf.CMsgClientGetAppOwnershipTicket{
		AppId: proto.Uint32(appID),
	})
	jobID := a.client.GetNextJobId()
	msg.SetSourceJobId(jobID)

	raw, err := a.sendJob(msg, jobID)
	if err != nil {
		return nil, err
	}
	return raw.(*AppOwnershipTicketResult), nil
}

// PICSGetAccessTokens 请求一批 app/package 的 PICS 访问令牌。
// 对应 SteamKit2 的 SteamApps.PICSGetAccessTokens(appIds, packageIds)。
func (a *Apps) PICSGetAccessTokens(appIDs, packageIDs []uint32) (*PICSTokensResult, error) {
	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientPICSAccessTokenRequest, &protobuf.CMsgClientPICSAccessTokenRequest{
		Appids:     appIDs,
		Packageids: packageIDs,
	})
	jobID := a.client.GetNextJobId()
	msg.SetSourceJobId(jobID)

	raw, err := a.sendJob(msg, jobID)
	if err != nil {
		return nil, err
	}
	return raw.(*PICSTokensResult), nil
}

// GetDepotDecryptionKey 请求指定 depot 的解密密钥。
// 对应 SteamKit2 的 SteamApps.GetDepotDecryptionKey(depotid, appid)。
func (a *Apps) GetDepotDecryptionKey(depotID, appID uint32) (*DepotKeyResult, error) {
	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientGetDepotDecryptionKey, &protobuf.CMsgClientGetDepotDecryptionKey{
		DepotId: proto.Uint32(depotID),
		AppId:   proto.Uint32(appID),
	})
	jobID := a.client.GetNextJobId()
	msg.SetSourceJobId(jobID)

	raw, err := a.sendJob(msg, jobID)
	if err != nil {
		return nil, err
	}
	return raw.(*DepotKeyResult), nil
}

// PICSGetChangesSince 请求自指定变更号以来的 PICS 变更列表。
// 对应 SteamKit2 的 SteamApps.PICSGetChangesSince(lastChangeNumber, sendAppChangelist, sendPackageChangelist)。
func (a *Apps) PICSGetChangesSince(lastChangeNumber uint32, sendAppChangelist, sendPackageChangelist bool) (*PICSChangesResult, error) {
	msg := protocol.NewClientMsgProtobuf(steamlang.EMsg_ClientPICSChangesSinceRequest, &protobuf.CMsgClientPICSChangesSinceRequest{
		SinceChangeNumber:      proto.Uint32(lastChangeNumber),
		SendAppInfoChanges:     proto.Bool(sendAppChangelist),
		SendPackageInfoChanges: proto.Bool(sendPackageChangelist),
	})
	jobID := a.client.GetNextJobId()
	msg.SetSourceJobId(jobID)

	raw, err := a.sendJob(msg, jobID)
	if err != nil {
		return nil, err
	}
	return raw.(*PICSChangesResult), nil
}

// sendJob 发送消息并等待对应 JobID 的响应（带超时）。
func (a *Apps) sendJob(msg protocol.IMsg, jobID protocol.JobId) (interface{}, error) {
	ch := make(chan interface{}, 1)
	a.mutex.Lock()
	a.jobs[jobID] = ch
	a.mutex.Unlock()

	a.client.Write(msg)

	select {
	case raw := <-ch:
		return raw, nil
	case <-time.After(30 * time.Second):
		a.mutex.Lock()
		delete(a.jobs, jobID)
		a.mutex.Unlock()
		return nil, errors.New("等待 Steam 响应超时")
	}
}