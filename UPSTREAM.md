# 上游基线

本仓库是 [miracleEverywhere/dst-management-platform-api](https://github.com/miracleEverywhere/dst-management-platform-api)
与 [miracleEverywhere/dst-management-platform-web](https://github.com/miracleEverywhere/dst-management-platform-web)
的整合 Fork，由 [ljj13/dst-management-platform](https://github.com/ljj13/dst-management-platform) 维护。

## 固定基线

| 组件 | 上游仓库 | 精确 commit | 说明 |
| --- | --- | --- | --- |
| 后端 | `miracleEverywhere/dst-management-platform-api` | `5b69f0100f567845ca3fe9a36f236cfa1dbe4d68` | `master`，版本常量 `v3.2.3`，2026-11-24 |
| 前端 | `miracleEverywhere/dst-management-platform-web` | `2fc1d1306f3a13225f2c2a7d446306059846ecaf` | 2026-11-23 23:02，`添加全局封禁模组` |

前端 commit 的选择依据：后端最后一次更新 `embedFS/dist` 的提交为 `dd625fcd`（2026-11-23 23:17），
其产物包含 `bannedMods` chunk；上游前端随后在 `836d6d7`（2026-11-23 23:44）新增的
“该房间包含…屏蔽模组”文案尚未进入该产物。因此 `2fc1d13` 是与 `v3.2.3` 后端 `embedFS/dist`
最接近、可复现的源码基线。

后端 `v3.2.2` 标签指向 `13b414f4925f6edd640cae79ea90c7be2554df92`（2026-11-07）。
`v3.2.2..v3.2.3` 包含服务层拆分、游戏大厅、AI 诊断、全局模组屏蔽等变更；数据库变更通过
GORM `AutoMigrate` 增量执行，生产升级前仍应按 `TASK_REPORT.md` 做一致性备份。

## 目录与许可

- 后端源码：仓库根目录，许可见 `LICENSE`。
- 前端源码：`frontend/`，许可见 `frontend/LICENSE`（MIT，Copyright (c) 2024 Miracle）。
- 上游版权与来源信息保留在各目录的 `LICENSE`、`README.md`、`AGENTS.md` 及本文件。
- `embedFS/dist/` 与 `lite/dist-lite/` 是构建产物，不手工编辑；标准版由 `frontend/` 执行
  `pnpm run build` 后整体复制，Lite 版由 `pnpm run build:lite` 生成后整体复制。
