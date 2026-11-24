# DMP v3.2.3-foghub.1 原生监控增强与云端构建报告

- 生成时间：2026-11-24
- Fork：<https://github.com/ljj13/dst-management-platform>
- 分支：`feat/native-monitor`
- 功能与发布提交：`9d72f3cfef32acec76dedfe567ffbb7b0d01245f`
- 报告提交：本文件所在的分支 HEAD
- 发布标签：`v3.2.3-foghub.1`
- 生产部署：**未执行**（未获得替换运行中 DMP 的授权）

## 1. 结论

P0-P5 的源码、构建、测试与发布工作已实际完成：

| 阶段 | 状态 | 结果 |
| --- | --- | --- |
| P0 基线与 Monorepo | 完成 | 上游前端精确版本已纳入 `frontend/`，统一构建脚本与来源记录已提交 |
| P1 GitHub Actions | 完成 | CI 与 Release 均真实运行成功，产物已发布 |
| P2 后端原生监控 | 完成 | 5 秒采集、进程识别、历史降采样、只读 API 已实现 |
| P3 前端监控页 | 完成 | 原系统监控页增强，桌面/移动端、明暗主题、中英文均实测 |
| P4 验证 | 部分完成 | API、权限、进程、断线、内存、静态资源已验证；未使用真实 DST 存档做集成测试 |
| P5 版本与发布 | 完成 | `v3.2.3-foghub.1` Release、`dmp.tgz`、`dmp-lite.tgz`、`SHA256SUMS` 已发布 |

生产服务器没有被连接、重启或替换。本文第 9 节给出部署与回滚方案，供获得授权后执行。

## 2. 上游基线与来源

### 2.1 代码基线

| 项目 | 值 |
| --- | --- |
| Fork 后端基线 | `5b69f0100f567845ca3fe9a36f236cfa1dbe4d68`（上游 `master`，Version `v3.2.3`，2026-11-24） |
| 生产版本参考 | `v3.2.2` = `13b414f4925f6edd640cae79ea90c7be2554df92`（2026-11-07） |
| 前端源码基线 | `2fc1d1306f3a13225f2c2a7d446306059846ecaf`（上游前端，2026-11-23 23:02） |
| Monorepo 整合提交 | `build: integrate upstream web frontend into monorepo` |
| 功能与发布提交 | `9d72f3cfef32acec76dedfe567ffbb7b0d01245f` |

前端 commit 的选择依据：后端最后一次“更新静态资源”为 `dd625fcd`（2026-11-23 23:17），其 `embedFS/dist` 已包含 `bannedMods` 资源，但不包含前端 `836d6d7`（23:44）新增的“该房间包含…屏蔽模组”文案。因此 `2fc1d13` 是与后端 `v3.2.3` 静态资源基线匹配的完整源码版本。详细来源记录见仓库根目录 `UPSTREAM.md`。

### 2.2 v3.2.2 → v3.2.3 差异与数据库兼容性

`v3.2.2..5b69f010` 共 **576 files changed, 23,544 insertions(+), 12,418 deletions(-)**，包含 service 层重构、游戏大厅、AI 诊断、全局模组屏蔽等变化。

数据库迁移检查结论：

- 使用 GORM `AutoMigrate`，启动时自动补齐表和列。
- 新增表：`banned_mods`、`diagnosis_records`、`diagnosis_action_records`。
- 新增系统设置键：`cloud_key`、`ai_diagnosis_setting`。
- 未发现删除表、删除列、重命名列或破坏性类型变更。
- 现有表模型未发生结构变更（`git diff v3.2.2..HEAD -- database/models` 仅显示新增模型与系统键）。
- 升级仍需先备份数据库；`AutoMigrate` 只保证增量兼容，不能替代备份。

### 2.3 许可证与生成物

- 后端许可证保留在仓库根目录 `LICENSE`。
- 前端许可证保留在 `frontend/LICENSE`（MIT，Copyright 2024 Miracle）。
- 未提交 `node_modules`、Go 构建缓存、生产数据库、游戏存档、密钥或日志。
- `embedFS/dist` 与 `lite/dist-lite` 是仓库原本就跟踪的发布生成物，本次提交了由 `pnpm run build`/`build:lite` 生成的新版本；没有手工修改压缩后的 JS/CSS。

## 3. Monorepo 与本地构建

### 3.1 主要改动

- 新增 `frontend/`，完整导入匹配基线的上游前端源码（584 个文件）。
- `frontend/package.json`：`name` 改为 `dst-management-platform-web`，版本 `3.2.3-foghub.1`，增加 `packageManager: pnpm@10.33.3` 与 `engines.node >=22`。
- 重写根 `Makefile`：
  - `FRONTEND_DIR := frontend`
  - `pnpm --dir frontend install --frozen-lockfile`
  - `pnpm --dir frontend run build` / `build:lite`
  - `node scripts/copy-dir.mjs frontend/dist embedFS/dist`
  - `node scripts/copy-dir.mjs frontend/dist-lite lite/dist-lite`
  - `CGO_ENABLED=0 go build`
- 新增跨平台复制脚本 `scripts/copy-dir.mjs`。
- 更新 `AGENTS.md`，说明 Monorepo、前端构建、生成物和验证规则。
- `.gitignore` 增加前端构建目录、Go 缓存、`.tools/`、`.smoke-data/` 等。
- 修复 `embedFS/embed.go` 的跨平台路径问题：原先 `filepath.Join` 在 Windows 生成反斜杠路径，`embed.FS.ReadFile` 失败并导致启动 panic；改为 `path.Join` 后 Windows 与 Linux 均正常。

### 3.2 本地构建验证

- `gofmt -l`：无输出。
- `go vet ./...`：通过。
- `go vet -tags=lite ./...`：通过。
- `make codes-verify`：通过（同步 169 个错误码文档条目）。
- 前端 `pnpm run lint`：通过。
- 标准前端 `pnpm run build`：通过（Vite 8.3.4）。
- Lite 前端 `pnpm run build:lite`：通过。
- 标准 `dmp.exe`：43,879,424 bytes。
- Lite `dmp-lite.exe`：8,290,816 bytes。
- `dmp.exe -v`：`v3.2.3-foghub.1`。

本地构建仅用于开发验证；正式发布产物由 GitHub Actions 在 `ubuntu-latest` 上重新构建。

## 4. GitHub Actions

### 4.1 工作流结构

- `.github/workflows/ci.yml`
  - 触发：所有分支 push、pull_request、workflow_dispatch。
  - 调用可复用的 `build.yml`。
  - `permissions: contents: read`，`concurrency` 取消同分支旧运行。
- `.github/workflows/build.yml`
  - checkout、pnpm 10.33.3、Node 22、Go（`go-version-file: go.mod`）。
  - pnpm 与 Go 缓存。
  - `pnpm install --frozen-lockfile`。
  - 前端 lint、标准构建、Lite 构建。
  - 复制并校验 `embedFS/dist` / `lite/dist-lite`。
  - `gofmt`、`make codes-verify`、`go vet ./...`、`go vet -tags=lite ./...`。
  - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` 构建 `dmp` 与 `dmp-lite`。
  - 打包 `dmp.tgz`、`dmp-lite.tgz`、`SHA256SUMS`。
  - 启动临时实例，执行注册、登录、monitor current/history/processes、静态资源、普通用户权限、无 token 权限冒烟。
  - 上传 artifact `dmp-linux-amd64`。
- `.github/workflows/release.yml`
  - 触发：`v*` 标签。
  - 调用 `build.yml`。
  - 校验标签与 `constant/system.go` 的 `Version` 一致。
  - 下载 artifact，`sha256sum -c` 校验。
  - 使用 `softprops/action-gh-release@v2` 和 `GITHUB_TOKEN` 发布 Release。
  - `permissions: contents: write`。
- 原 `.github/workflows/go.yml` 已删除，避免 tag 双发布与上游 GHCR 硬编码。Docker 构建不是发布必需项，未让 Docker 阻断 Linux amd64。

### 4.2 真实运行结果

| 项目 | 结果 |
| --- | --- |
| CI 运行 | <https://github.com/ljj13/dst-management-platform/actions/runs/45351359668> |
| CI 结果 | **success** |
| CI 时间 | 2026-11-24 07:52:26Z → 07:56:20Z |
| CI 提交 | `9d72f3cfef32acec76dedfe567ffbb7b0d01245f` |
| Release 运行 | <https://github.com/ljj13/dst-management-platform/actions/runs/45352019308> |
| Release 结果 | **success** |
| Release 时间 | 2026-11-24 07:57:32Z → 08:01:23Z |
| Release 标签 | <https://github.com/ljj13/dst-management-platform/releases/tag/v3.2.3-foghub.1> |
| Artifact | `dmp-linux-amd64`，24,591,657 bytes（CI 构建） |

CI 日志确认以下步骤均成功：前端 frozen install、lint、标准/Lite 构建、静态资源校验、gofmt、错误码文档校验、Go vet、Linux amd64 构建、打包、冒烟测试、artifact 上传。

## 5. P2 后端原生监控

### 5.1 代码位置

- `monitor/types.go`：采样结构、历史点、进程结构、JSON 字段。
- `monitor/collector.go`：单例采集器、5 秒采样、24 小时环形历史、降采样。
- `monitor/process.go`：进程枚举、CPU 差值、DST/DMP 识别、PID 复用处理。
- `service/platform/monitor.go`：业务层响应与参数校验。
- `app/platform/handler.go`：三个只读 handler。
- `app/platform/router.go`：路由注册。
- `server/server.go`：启动 `monitor.Start()`，退出时 `monitor.Stop()`。

### 5.2 采集内容

- CPU：整机使用率、每 vCPU、型号、物理/逻辑核心、MHz、Load1/5/15、User/System/Idle/IOWait/Steal/Nice/Irq/Softirq。
- 内存：Total/Used/Available/Free/Buffers/Cached/BuffCache/UsedPercent。
- Swap：Total/Used/Free/UsedPercent/Sin/Sout。
- 磁盘：分区容量、使用率、inode；设备级读写字节、读写次数、IOPS、队列深度、IO 时间。
- 网络：每网卡累计字节/包、实时 B/s、Mbps、MB/s；聚合时排除 loopback、docker、veth、br-、virbr 等虚拟接口。
- 进程：PID、名称、CPU%、RSS、内存占比、状态、运行时间、创建时间、角色、Shard、Cluster。
- 进程角色：`dmp`、`dst-master`、`dst-caves`、`dst`、`other`。

### 5.3 采样与历史

- 默认采样间隔：5 秒。
- 历史：24 小时环形缓冲，容量 17,280 点。
- 查询范围与目标点数：
  - `15m`：约 180 点；
  - `1h`：约 240 点；
  - `24h`：约 720 点。
- 超出目标点数时按桶降采样，返回 `downsampled: true` 与 `sampleIntervalSeconds`。
- 进程表最多保留 300 条（CPU top 200 + 内存补充 + 始终保留 DST/DMP），因此 `processPartial` 可能为 `true`；这是有界内存设计，不是采集失败。
- 采集错误写入 `warnings`，前端显示降级状态；不会用 0 或假数据填充。

### 5.4 API

新增路由均位于已有 `platform` 分组，复用 `middleware.TokenCheck()` 与 `middleware.AdminOnly()`：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/v3/platform/monitor/current` | 当前快照；首次采样前返回 `available:false,status:"collecting"` |
| GET | `/v3/platform/monitor/history?range=15m\|1h\|24h` | 历史点；非法范围返回 `code:5001` |
| GET | `/v3/platform/monitor/processes?sort=cpu\|memory\|name&order=asc\|desc&q=&limit=` | 进程查询；limit 上限 500 |

安全边界：

- 只读接口，没有终止进程、执行 Shell、修改系统或提权接口。
- 不返回环境变量、SSH 私钥、Klei Token、完整命令行或凭据。
- 进程结构只包含白名单字段；DST 命令行只在采集器内部解析 Shard/Cluster，不进入 API。
- 不额外开放端口，复用现有 DMP HTTP 服务。
- 监控 API 仅 admin 可访问；普通用户返回 `code:2`，无 token 返回 `code:420`。

## 6. P3 前端监控页

直接增强原 `系统监控` 页，没有新增另一套监控系统。主要文件：

- `frontend/src/views/platform/components/metrics.vue`
- `frontend/src/api/platform.js`
- `frontend/src/languages/lang-zh.js`
- `frontend/src/languages/lang-en.js`
- `frontend/src/components/scEcharts/index.vue`

功能：

- 8 张总览卡：CPU、内存、Swap、磁盘、上传、下载、运行时间、进程数。
- CPU 总览 + Load 双轴图；最多 4 张独立 vCPU 图，右上显示当前核心使用率；超过 4 核提示仍有核心未绘制，但 API 始终返回全部核心。
- Steal/IOWait 独立图表。
- 内存/Swap 历史曲线。
- 磁盘读写 MB/s。
- 网络上下行 Mbps，并保留 B/s、MB/s 字段供核对。
- DST/DMP 资源双轴图：Master/Caves CPU 与 RSS。
- 进程表：搜索、按 CPU/内存/名称排序、升降序、分页、角色 chip、CPU/内存/状态/运行时间。
- 时间范围：15 分钟、1 小时、24 小时。
- 明暗主题、移动端单列布局、中英文文案。
- 自动刷新：current 5 秒，history/processes 10 秒；页面隐藏时暂停；组件卸载清理定时器。
- 采集失败显示真实错误，不伪造曲线；`available:false` 显示采集中状态。
- 修复 `scEcharts` 未销毁实例的问题：卸载时移除 resize 监听并 `dispose`，避免长期切换页面导致图表内存增长。

## 7. P4 验证结果

### 7.1 API 与数据正确性

在本机 Windows 实例和 CI Linux 实例上验证：

| 验证项 | 结果 |
| --- | --- |
| 首次注册 | `code:200`，首个用户为 admin |
| 登录 | 返回 JWT 字符串 |
| current | `code:200`、`available:true`、`status:"ok"` |
| CPU 核心 | 本机 32 核，`perCore` 长度 32 |
| 进程 | `processes` 300 条、`processTotal` 约 484、`partial:true` |
| 历史 | 15m/1h/24h 均 `available:true`，采样间隔 5 秒 |
| 非法范围 | `code:5001` |
| 进程排序 | `sort=memory&order=desc&limit=5` 返回 5 条，排序正常 |
| 静态页面 | `/` 200，`/assets/metrics-*.js` 200 |
| 并发客户端 | 两个并发 current 请求返回相同 snapshot timestamp，浏览器不触发重复采样 |
| 断线恢复 | 停服后显示 `监控数据加载失败: Network Error`；重启后刷新恢复 |
| 单位 | `upBps=225154.31`、`upMbps=1.8`、`upMBps=0.21` 与换算一致 |
| 敏感字段 | 进程 JSON 不含 `cmdline`/`environment` |

### 7.2 权限

| 场景 | 结果 |
| --- | --- |
| admin 访问 monitor | `code:200` |
| 普通用户访问 monitor | `code:2`（权限不足） |
| 普通用户访问 overview | `code:2` |
| 无 token 访问 monitor | HTTP 200，JSON `code:420`（Token 认证失败） |

权限判断看 JSON `code`，不能只看 HTTP 状态；这是项目的统一响应封装行为。

### 7.3 进程识别与重启

使用临时编译的假进程验证，不使用真实 DST 存档或玩家数据：

- 进程名 `dontstarve_dedicated_server_nullrenderer_x64.exe`，参数 `-shard Master -cluster TestCluster`：识别为 `dst-master`，Shard `Master`，Cluster `TestCluster`。
- 结束该进程后 7 秒查询：`matched=0`，确认退出检测。
- 以同集群、`-shard Caves` 重启：识别为 `dst-caves`，新 PID 23312、新 CreateTime，确认 PID 复用/重启处理。
- 当前 DMP 进程：识别为 `dmp`，PID 45360。
- 清理临时进程与临时目录。

Windows 上 gopsutil 进程状态返回 `unknown`，属平台限制；Linux 下状态字段正常。

### 7.4 浏览器 UI

使用真实浏览器访问 `http://127.0.0.1:18080/`，登录 admin 后进入“系统监控”：

- 桌面端完整渲染 8 张总览卡、CPU/Load、4 张 vCPU 图、Steal/IOWait、内存/Swap、磁盘 IO、网络、DST/DMP 资源、进程表。
- 暗色主题：设置 `localStorage.dmp-global.theme='dark'` 后重载，页面和 ECharts 均正常。
- 移动端：iPhone 13 视口，卡片单列、控件换行、图表自适应。
- 英文：设置语言为 `en` 后，`Updated`、`Time Range`、`Auto refresh`、`Refresh`、`CPU`、`Memory`、`Swap`、`Disk`、`Upload` 等文案正常。
- 截图：
  - `C:\Users\commender\.dsh\profiles\desktop\dsh-browser-shots\2026-11-24T07-19-35-343Z-kdl4.png`
  - `C:\Users\commender\.dsh\profiles\desktop\dsh-browser-shots\2026-11-24T07-20-10-772Z-7r3w.png`
  - `C:\Users\commender\.dsh\profiles\desktop\dsh-browser-shots\2026-11-24T07-37-53-569Z-e4xw.jpg`
  - `C:\Users\commender\.dsh\profiles\desktop\dsh-browser-shots\2026-11-24T07-38-15-454Z-eegp.jpg`

### 7.5 内存与长时间刷新

- 连续 15 次刷新：JS heap 约增加 19 MB。
- 再连续 15 次刷新：回落约 14 MB。
- 未出现单调增长；历史点有上限，图表实例在页面卸载时销毁。
- 该测试是本机浏览器手动采样，不是自动化泄漏基准；后续可在 Playwright 中固化。

### 7.6 CI 隔离环境

`build.yml` 在 GitHub 托管的全新 Ubuntu 实例中执行完整构建和 API 冒烟，不依赖生产数据库、真实 DST 存档或玩家数据；该运行已成功。

## 8. P5 版本、发布与升级安全

### 8.1 版本

- `constant/system.go`：`Version = "v3.2.3-foghub.1"`
- `frontend/src/config/index.js`：同步显示版本。
- `frontend/package.json`：`3.2.3-foghub.1`。
- `dmp.exe -v` 与 Release 标签一致。
- `release.yml` 在发布前校验标签与 `constant/system.go`，不一致则失败。

### 8.2 Release 产物

Release：<https://github.com/ljj13/dst-management-platform/releases/tag/v3.2.3-foghub.1>

| 文件 | 大小 | SHA-256 |
| --- | --- | --- |
| `dmp.tgz` | 18,702,237 bytes | `5a09e5aa1d893dc7a0268167c321ed463d8d8d680f4f6260f51f8cf129b28025` |
| `dmp-lite.tgz` | 5,931,766 bytes | `d1ef60a7e461a7fce9a5145451e9663c45c4b5e52fda2992a7933cb9f37f0192` |
| `SHA256SUMS` | 153 bytes | `5e15a1d1308cb869c6b1923070279d70fda1a5d1ed60be1e2f9f127d5408d621` |

已下载 Release 附件并用 `Get-FileHash` 复核，两个 `.tgz` 的哈希与 `SHA256SUMS` 一致。`dmp.tgz` 内含 Linux amd64 `dmp`；`dmp-lite.tgz` 内含 `dmp-lite`、`dist-lite`、安装/更新脚本与 Lite `run.sh`。

说明：CI 验证构建与 Release 重建是两次独立构建，产物哈希不同（Vite/Go 构建非逐字节可复现）。发布与校验以 Release 附件中的 `SHA256SUMS` 为准。

### 8.3 run.sh 定制更新保护

`run.sh` 已修改：

- `DMP_GITHUB_HOME_URL`、`DMP_GITHUB_API_URL`、`SCRIPT_GITHUB`、`DMP_HOME` 全部指向 fork `ljj13/dst-management-platform`。
- `get_latest_version`：
  - 检查空值与 `null`；
  - 拒绝非 `*-foghub.*` 标签，避免上游官方版本覆盖定制版。
- `install_dmp`：
  - 读取 Release 的 `.tag_name`；
  - 再次拒绝非 foghub 定制发布；
  - 仍下载 `dmp.tgz` 并校验 GitHub asset digest；
  - 保留原有安装、启动、停止、脚本更新能力。
- 未加入自动更新 DST 本体、修改游戏端口或触碰 Klei Token 的逻辑。

版本比较已用 `sort -V` 验证：

- `v3.2.3` < `v3.2.3-foghub.1`
- `v3.2.3-foghub.1` < `v3.2.3-foghub.2`
- `v3.2.3-foghub.1` < `v3.2.4`
- 上游更新为 `v3.2.4` 时，`*-foghub.*` 守卫会阻止自动覆盖，需人工发布新的 `v3.2.4-foghub.1`。

`run.sh` 已通过 WSL `bash -n` 语法检查。

## 9. 部署与回滚方案（未执行）

生产环境信息来自任务描述：腾讯云 Lighthouse Ubuntu 26.04.1 x86-64、4 vCPU、3.6 GiB RAM、2 GiB Swap、39 GB 盘，原 DMP `v3.2.2`，已运行 DST Master/Caves，域名 `dst.foghub.org`。

**没有获得替换运行中 DMP 的授权，因此以下步骤均未在生产执行。**

### 9.1 部署前检查

1. 确认 DMP 实际安装目录、二进制路径、启动方式（systemd/run.sh/screen）、监听端口、`-dbpath` 数据库位置。
2. 确认 DST Master/Caves 的 screen/进程状态与当前玩家数量。
3. 备份：
   - 当前 `dmp` 二进制；
   - `run.sh` 及环境配置；
   - 数据库文件（SQLite 使用一致性备份，如 `sqlite3 dmp.db ".backup ..."`，不要直接复制正在写入的库）；
   - 不触碰游戏存档、MOD、Klei Token、玩家数据。
4. 校验 `dmp.tgz` 的 SHA-256 与 Release `SHA256SUMS` 一致。
5. 在临时目录解压并运行 `./dmp -v`，确认输出 `v3.2.3-foghub.1`。

### 9.2 部署步骤（授权后）

1. 停止 DMP 服务（只停 DMP，不重启 Master/Caves）。
2. 再次确认数据库一致性和备份文件可读。
3. 用新二进制替换旧二进制，保留旧文件为 `dmp.v3.2.2.bak`。
4. 用原启动参数启动，确认：
   - 进程正常；
   - 日志无迁移 panic；
   - 登录、平台页、系统监控 API 正常；
   - Master/Caves 进程未被误杀。
5. 验证管理员登录、`/v3/platform/monitor/current`、历史、进程表、普通用户权限拒绝。
6. 观察至少一个 5 秒采样周期和一次 1 分钟旧 metrics 采集。

### 9.3 回滚

出现启动失败、数据库迁移异常、监控页不可用或 DMP 影响 DST 控制时：

1. 停止新 DMP。
2. 恢复旧 `dmp` 二进制。
3. 如数据库迁移导致不兼容，使用部署前一致性备份恢复数据库。
4. 按原方式启动旧版本。
5. 确认 Master/Caves 仍在运行、玩家可连接、DMP 页面可登录。
6. 保留日志与失败现象，不在生产反复试错。

## 10. 已知限制与剩余问题

1. **未做生产部署**：无授权，未连接生产服务器，未验证真实 4 vCPU/3.6 GiB 环境下的页面与采样开销。
2. **未用真实 DST 进程测试**：使用临时假进程验证角色识别、退出、重启与 PID 复用；未加载真实存档。
3. **进程列表有界**：最多 300 条，`processPartial` 可能为 `true`；这是为了避免每次请求枚举全部进程。
4. **Windows 状态字段受限**：gopsutil 在 Windows 返回 `unknown`；Linux 正常。
5. **网卡速率仅 Linux**：`linkSpeedMbps` 读取 `/sys/class/net/*/speed`，其他平台为空。
6. **网络聚合排除虚拟接口**：按名称前缀过滤，极端自定义命名可能被误判；单网卡明细仍返回全部接口。
7. **构建非逐字节可复现**：CI 验证构建与 Release 构建哈希不同；使用 Release `SHA256SUMS` 校验。
8. **Playwright 未固化为自动测试**：浏览器验证为人工/工具驱动；项目策略禁止新增 `_test.go`，未添加 Go 测试文件。
9. **CI 仅 Linux amd64**：没有 macOS/ARM 发布矩阵；任务要求的生产目标为 Linux amd64。
10. **Docker 未纳入必需发布路径**：避免 Docker 构建阻断 Linux amd64；原上游 GHCR 硬编码已移除。
11. **Release 的 latest 语义**：`run.sh` 使用 GitHub `/releases/latest`；如果未来把定制版标记为 prerelease，latest 可能指向旧版本，需要在发布流程中保持非 prerelease 或改用 releases 列表筛选。
12. **网络字段大小写**：`upMbps` 与 `upMBps` 仅大小写不同，JSON/JavaScript 合法且数值分别正确；PowerShell `ConvertFrom-Json` 这类大小写不敏感解析器会把二者视为同名，建议这类客户端改用 `upBps`/`downBps`。

## 11. 关键文件索引

- 后端监控：`monitor/types.go`、`monitor/collector.go`、`monitor/process.go`
- 监控业务与路由：`service/platform/monitor.go`、`app/platform/handler.go`、`app/platform/router.go`
- 服务启动：`server/server.go`
- 前端页面：`frontend/src/views/platform/components/metrics.vue`
- 前端 API：`frontend/src/api/platform.js`
- 前端图表组件：`frontend/src/components/scEcharts/index.vue`
- 工作流：`.github/workflows/build.yml`、`.github/workflows/ci.yml`、`.github/workflows/release.yml`
- 构建脚本：`Makefile`、`scripts/copy-dir.mjs`
- 升级脚本：`run.sh`
- 来源记录：`UPSTREAM.md`
- 开发记录：`AGENTS_WORK.md`（按仓库规则被 `.gitignore` 忽略，不进入提交）

## 12. 验收对照

| 要求 | 证据 |
| --- | --- |
| 上游前端完整源码进入 Monorepo | `frontend/` 584 文件，commit `2fc1d13` 记录于 `UPSTREAM.md` |
| 统一构建 | 根 `Makefile`、`scripts/copy-dir.mjs`；CI 真实构建成功 |
| 不手改压缩资源 | 只通过 `pnpm run build` 生成并整体复制 dist |
| 5 秒原生采集 | `monitor.IntervalSeconds=5`；history `sampleIntervalSeconds:5` |
| 每 vCPU/Steal/IOWait | API `perCore` 32 项；前端 4 张独立 vCPU 图 + Steal/IOWait 图 |
| 进程单核 CPU | 进程 CPU 用两次 `Times` 差值/墙钟，单核满载约 100% |
| 进程排序/搜索/角色 | API processes 与前端进程表实测 |
| DST/DMP 识别 | 假进程 Master/Caves、DMP 实测；退出/重启/PID 处理实测 |
| 权限 | admin 200、普通用户 `code:2`、无 token `code:420` |
| 敏感信息 | 进程 JSON 无 cmdline/environment；无 Shell/终止接口 |
| 历史与降采样 | 24h 环形 17,280 点；15m/1h/24h 目标 180/240/720 点 |
| 前端双语/主题/移动端 | 浏览器实测中英文、暗色、iPhone 13 |
| Actions 真实成功 | CI run 45351359668、Release run 45352019308 均 success |
| Linux amd64 发布 | Release `v3.2.3-foghub.1`，`dmp.tgz` + `SHA256SUMS` |
| 不部署生产 | 未连接、未替换、未重启生产 DMP |

---

本报告是本次任务的唯一交付报告；所有“未执行”的项目均未标记为成功。
