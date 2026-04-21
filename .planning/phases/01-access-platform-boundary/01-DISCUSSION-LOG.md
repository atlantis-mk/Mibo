# Phase 1: Access & Platform Boundary - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-21
**Phase:** 01-access-platform-boundary
**Areas discussed:** 初始化入口

---

## 初始化入口

| Option | Description | Selected |
|--------|-------------|----------|
| 强制 Setup | 未完成初始化时，除 `/setup` 外都不能进入。最符合当前路由结构，也最不容易出现半配置状态。 | ✓ |
| 允许只读壳层 | 先能进应用外壳，但核心区域提示去完成 setup。体验更软，但状态分支会更多。 | |
| 只拦首个管理员 | 只在没有管理员账号时强拦；有账号后允许先登录，再补媒体源和媒体库。 | |

**User's choice:** 强制 Setup
**Notes:** 初始化前保持强门禁，未完成 setup 不进入主应用。

---

## 初始化入口 / 完成条件

| Option | Description | Selected |
|--------|-------------|----------|
| 账号即可进入 | 只要创建了管理员账号就能进应用，媒体源和媒体库后面再补。与当前 `can_enter_app` 语义一致。 | ✓ |
| 账号+媒体源 | 至少要创建管理员账号和一个媒体源，避免进应用后还是空壳。 | |
| 账号+源+库 | 必须完成管理员账号、媒体源、媒体库三步，才算 setup 完成。更符合“进入后就能开始用”的产品感受。 | |

**User's choice:** 账号即可进入
**Notes:** 用户希望更早放行进入应用，媒体源和媒体库可在后续补齐。

---

## 初始化入口 / 首屏落点

| Option | Description | Selected |
|--------|-------------|----------|
| 回设置引导 | 进入主应用后，直接落到一个明确的配置引导页/空状态，继续补媒体源和媒体库。 | ✓ |
| 进入首页空状态 | 允许进正常首页，但首页展示强提示，引导去创建媒体源和媒体库。 | |
| 进入设置页 | 登录后直接落到 settings/source-library 管理区域，更偏管理后台式体验。 | |

**User's choice:** 回设置引导
**Notes:** 已创建账号但媒体配置未完成时，首屏不应是媒体首页空壳。

---

## 初始化入口 / Setup 形态

| Option | Description | Selected |
|--------|-------------|----------|
| 分步向导 | 继续保留 wizard 体验，把首次接入做成明显的 1-2-3 步流程。 | ✓ |
| 配置中心页 | 改成一个可自由跳转的设置中心，每块配置独立完成。 | |
| 向导 + 可返回编辑 | 默认仍是分步向导，但完成或中断后可以回到配置页面继续补全。 | |

**User's choice:** 分步向导
**Notes:** Phase 1 不改成后台式配置中心，保持首次接入的顺序式体验。

---

## the agent's Discretion

- 具体 UI 布局、文案和 loading/error 呈现
- 配置引导落点挂载在首页壳层还是独立引导页中的具体实现

## Deferred Ideas

None.
