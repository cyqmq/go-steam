# go-steam（ASF-Go 自维护分支）

本仓库是 [Philipp15b/go-steam/v3](https://github.com/Philipp15b/go-steam) 的自维护分支，
由 ASF-Go 项目维护，用于补充原版停更后缺失的 Steam 协议功能。

## 与原版的差异

### 新增：SteamApps handler（`apps.go`）

对应 SteamKit2 的 `Steam/Handlers/SteamApps`，当前实现：

- `Client.Apps.RequestFreeLicense(appID)` / `RequestFreeLicenseMulti(appIDs)`
  - 请求 appID 的免费许可（免费随选游戏激活）
  - 实现原理（扒自 SteamKit2）：
    1. 构造 `CMsgClientRequestFreeLicense` protobuf 消息（`EMsg_ClientRequestFreeLicense`）
    2. `SetSourceJobId(jobID)`（SteamKit2：`request.SourceJobID = Client.GetNextJobID()`）
    3. `Client.Write(msg)` 发送
    4. 通过 JobID 匹配 `EMsgClientRequestFreeLicenseResponse` 响应（SteamKit2：`AsyncJob<FreeLicenseCallback>`）
  - 返回 `FreeLicenseResult{Result, GrantedApps, GrantedPackages}`

- `Client.Apps.CheckAppBetaPassword(appID, password)`
  - 提交 beta 密码，返回可用 beta 分支及其加密密钥
  - 对应 SteamKit2：`SteamApps.CheckAppBetaPassword(app, password)` → `CheckAppBetaPasswordCallback`
  - 返回 `CheckAppBetaPasswordResult{Result, BetaPasswords[]}`

- `Client.Apps.GetAppOwnershipTicket(appID)`
  - 请求应用所有权票据
  - 对应 SteamKit2：`SteamApps.GetAppOwnershipTicket(appid)` → `AppOwnershipTicketCallback`
  - 返回 `AppOwnershipTicketResult{Result, AppID, Ticket}`

- `Client.Apps.PICSGetAccessTokens(appIDs, packageIDs)`
  - 请求一批 app/package 的 PICS 访问令牌
  - 对应 SteamKit2：`SteamApps.PICSGetAccessTokens(appIds, packageIds)` → `PICSTokensCallback`
  - 返回 `PICSTokensResult{AppAccessTokens, AppDeniedTokens, PackageAccessTokens, PackageDeniedTokens}`

- `Client.Apps.GetDepotDecryptionKey(depotID, appID)`
  - 请求指定 depot 的解密密钥
  - 对应 SteamKit2：`SteamApps.GetDepotDecryptionKey(depotid, appid)` → `DepotKeyCallback`
  - 返回 `DepotKeyResult{Result, DepotID, EncryptionKey}`

- `Client.Apps.PICSGetChangesSince(lastChangeNumber, sendAppChangelist, sendPackageChangelist)`
  - 请求自指定变更号以来的 PICS 变更列表
  - 对应 SteamKit2：`SteamApps.PICSGetChangesSince(lastChangeNumber, sendAppChangelist, sendPackageChangelist)` → `PICSChangesCallback`
  - 返回 `PICSChangesResult{CurrentChangeNumber, SinceChangeNumber, ForceFullUpdate, AppChanges, PackageChanges}`

## 维护方式

- ASF-Go 通过 `replace github.com/Philipp15b/go-steam/v3 => ../go-steam` 使用本仓库
- 模块路径保持 `github.com/Philipp15b/go-steam/v3` 不变，避免改动上层 import

## 测试

```bash
go test ./...
```

## 参考

- 原版：https://github.com/Philipp15b/go-steam
- SteamKit2（C# 协议参考）：https://github.com/SteamRE/SteamKit
- SteamKit2 SteamApps 源码：`SteamKit2/Steam/Handlers/SteamApps/SteamApps.cs`