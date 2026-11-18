# Clablings 题库设计文档
> **已被取代（2026-11-18）**：本文档中关于双语内容、`--lang`、`CLABLINGS_LANG`、`config.language`、`{en, zh}` 映射和语言回退的约定，已由 `2026-11-18-chinese-only-design.md` 取代；其余设计作为历史记录保留。

> 日期：2026-09-19
> 状态：已与用户逐节确认
> 依据：`clablings/docs/superpowers/specs/2026-09-17-clablings-design.md`、`2026-09-19-clablings-watch-ux-design.md`
> 目标：以内容（题库）为中心设计可长期自用并对开源发布的学习型实验库

## 1. 背景与目标

Clablings 现有引擎能力已闭环（`list/watch/run/reset/test/debug`、声明式断言、两级重置、C 题构建下发、headless 跑批、实例命名隔离），但题库只有三题、以"故障→修复"为主，缺少"为什么协议这么设计"的理解层，导致"看完文档就忘"的问题没有被解决。

本文档定义题库的信息架构、学习模型、内容 schema、多语言、出题与校准流程，以及由内容需求推导出的引擎/工具缺口。**原则：题库是产品本体，引擎能力随题库需求演进。**

## 2. 已锁定的决策

| # | 决策 |
|---|---|
| 1 | 面向自用 + 开源发布；先设计好题库，引擎跟着题库需求走 |
| 2 | 学习链路：`Why（问题）→ What（心智模型）→ How（关键机制）→ Quiz（自检）→ Do（实验）→ Read（RFC 精读切片）` |
| 3 | 章节结构：主题章（一个协议/主题、一个拓扑、多题难度递进）+ 阶段性综合实战章 |
| 4 | 题型：运行态 `config-fix`、文件驱动 `config-fix`、`code-implement`、**`quiz`（单选/多选，无容器，门控推进）** |
| 5 | 多语言：内容**默认英文**，支持 `en`/`zh` 切换；CLI 文案恒英文 |
| 6 | YAML 人类文案用内联多语言字段（纯字符串=英文，或 `{en, zh}`），缺失回退英文 |
| 7 | concept 正文用 `concept.<lang>.md`；RFC 阅读切片用结构化 `reading.<lang>.yaml` |
| 8 | 语言选择：`--lang` > `CLABLINGS_LANG` > 配置 `language` > 默认 `en` |
| 9 | 交换底座用真实 NOS：SR Linux（主），cEOS/SONiC 可选；堆叠仅讲概念，MLAG 重点讲并给出 EVPN-MH 现代替代 |
| 10 | 里程碑：M1 交换主线 → M2 Overlay（VXLAN/Geneve/BGP/EVPN）→ M3 L3 路由主线 |
| 11 | 元数据含 `difficulty/tags/estimated_minutes/prerequisites/objectives`，并提供 `clablings lint` |
| 12 | 断言新增 `wait`：轮询至通过或超时，支撑 STP/路由等收敛类判定 |
| 13 | 工具链新增 `make install` / `make uninstall`，默认安装到 `~/.local/bin` |

## 3. 学习模型（每章统一）

每章围绕 **learning objectives**（3–5 条，可度量）组织：

1. **Why（问题）**：先给出会失败的朴素方案或真实痛点，制造动机。
   例：静态路由断链不自愈、双上行无法同时使用、二层环路广播风暴。
2. **What（心智模型）**：一句话模型。
   例：STP 用"逻辑上剪断环路、物理上保留冗余"；MLAG 让两台交换机对外表现为一台。
3. **How（关键机制）**：只讲与本章实验相关的字段/定时器/状态机，并指出对应失败现象。
4. **Quiz（自检）**：单选/多选，答错显示解析，全对才推进——把"看完就忘"变成主动回忆。
5. **Do（实验）**：现有实验题，故障注入 + 学员修复/实现 + 断言判定。
6. **Read（精读切片）**：只给"RFC 编号 + 精确小节 + 我们的导读 + 一个带读问题"，不内嵌整篇 RFC。

## 4. 里程碑与章节路线

### M1 交换主线（首个里程碑）

| 章 | 主题 | 底座 | 重点 |
|---|---|---|---|
| 01 | L2 基础与 ARP | Linux | 二层转发、ARP/免费ARP/代理ARP（快速回顾） |
| 02 | VLAN 与 Trunk | Linux bridge / SR Linux | access/trunk、native VLAN、SVI 与 VLAN 间路由 |
| 03 | STP/RSTP | SR Linux（可先用 Linux bridge 打底） | 根桥选举、路径代价、端口角色/状态、拓扑变更、edge/guard |
| 04 | 链路聚合 LACP | SR Linux ↔ SR Linux / ↔ Linux | 静态 vs LACP、802.3ad、哈希与负载分担、min-links |
| 05 | MLAG（重点） | SR Linux EVPN-MH 或 cEOS vPC | 双归、peer-link、orphan 口、防脑裂；对比"堆叠"（仅概念） |

### M2 Overlay

| 章 | 主题 | 底座 | 重点 |
|---|---|---|---|
| 06 | VXLAN 数据面 | Linux / SR Linux | VTEP/VNI、封装、underlay/overlay、单播/组播、FDB |
| 07 | Geneve | Linux | 与 VXLAN 差异、TLV 选项、典型用途（OVN） |
| 08 | BGP 基础 | FRR | eBGP/iBGP、报文与状态机、路径属性、选路 |
| 09 | EVPN 控制面 | FRR + SR Linux | MAC/IP/IMET 路由、anycast gateway、EVPN-MH（MLAG 的开放替代） |
| 10 | 综合实战 | SR Linux + Linux | VLAN + LACP + MLAG/EVPN-MH + VXLAN 串联 |

### M3 L3 路由主线（可与 M2 交叉）

| 章 | 主题 | 底座 | 重点 |
|---|---|---|---|
| 11 | 静态路由与转发 | Linux/FRR | 路由表、最长前缀、转发（可吸纳现有 route-fix） |
| 12 | OSPF | FRR | 邻居/链路状态/收敛（可吸纳现有 ospf-mtu） |
| 13 | BGP 进阶 | FRR | 路由策略、前缀过滤、社区、多宿主 |
| 14 | VRRP/HA | FRR | 网关冗余、优先级与抢占、与 LACP/MLAG 的关系 |

### 现有三题的迁移

`01-ospf-mtu` → M3 第 12 章；`03-route-fix` → M3 第 11 章；`02-arp-responder` → M1 第 01 章（或作为 code-implement 例题保留）。迁移时补齐 concept/reading/quiz/objectives 与多语言。

## 5. 章节解剖与文件结构

```
chapters/<chapter-id>/
  chapter.yaml              # 元数据；topology 可选（纯概念/quiz 章可为空）
  concept.en.md             # 必填；Why/What/How
  concept.zh.md             # 可选；缺失回退英文
  reading.en.yaml           # 结构化 RFC 阅读切片
  reading.zh.yaml
  topology.clab.yml         # 可选
  Makefile                  # debug: clablings debug --sudo
  exercises/<exercise-id>/
    exercise.yaml
    README.en.md
    README.zh.md
```

综合实战章可含多个拓扑/多阶段，仍沿用同一章目录约定（`topology` 指向主拓扑，阶段差异由题目 `files`/`reload` 与 setup/teardown 表达）。

## 6. 内容 Schema

### 6.1 多语言字段（`LocalizedString`）

在 `internal/content` 定义类型，`UnmarshalYAML` 接受：
- 纯标量：视为默认语言（英文）文本；
- 映射：`{en: ..., zh: ...}`（键为 BCP-47 子集 `en`/`zh`）。

解析时按当前 locale 取值，缺失回退 `en`，再缺失回退任意可用值。人类文案字段（`title`、`name`、`hint`、`hints`、`prompt`、选项文本/解析、`intro`、`question`）均使用该类型；结构化字段（`cmd`、`match`、`files`、`reload`）与语言无关。

### 6.2 `chapter.yaml`

```yaml
id: 03-stp
title: {en: STP/RSTP, zh: STP/RSTP}
difficulty: 3
tags: [l2, stp, switching]
estimated_minutes: 30
prerequisites: [02-vlan]
objectives:
  en:
    - Explain why a layer-2 loop breaks an Ethernet network
    - Predict the root bridge and port roles for a given topology
  zh:
    - 解释二层环路为何会瘫痪以太网
    - 根据拓扑预测根桥与端口角色
topology: topology.clab.yml   # 可选
deploy: {timeout: 180s}
```

### 6.3 `exercise.yaml`

```yaml
id: 01_root_election
title: {en: Root bridge election, zh: 根桥选举}
type: config-fix          # config-fix | code-implement | quiz
difficulty: 2
timeout: 30s
hints:
  - {en: Lower bridge priority wins, zh: 桥优先级越低越优先}
  - {en: Compare the root path cost, zh: 比较到根的开销}
# config-fix / code-implement 字段同现状：files / reload / build / run
assertions:
  - name: {en: br0 is root bridge, zh: br0 成为根桥}
    type: exec
    layer: control-plane
    node: sw1
    cmd: "bridge -j stp show"
    wait: 20s            # 新增：轮询至匹配或超时
    match:
      json: "$.root_bridge"
      equals: "sw1"
    hint: {en: Set a lower priority, zh: 调低优先级}
```

### 6.4 Quiz 题

```yaml
id: 01_why_stp
title: {en: Why STP?, zh: 为什么需要 STP？}
type: quiz
questions:
  - prompt: {en: What happens first in a redundant L2 loop?, zh: 冗余二层环路首先会发生什么？}
    multiple: false
    options:
      - text: {en: Broadcast storm, zh: 广播风暴}
        correct: true
        explanation:
          en: An Ethernet frame with an unknown destination is flooded; with a loop it circulates forever and multiplies.
          zh: 目的未知的以太网帧会被泛洪；存在环路时会在环里永远循环并不断复制，形成广播风暴。
      - text: {en: The link is blocked automatically, zh: 链路会自动阻塞}
        correct: false
        explanation:
          en: Blocking is precisely what STP introduces; without it the loop floods.
          zh: 阻塞正是 STP 才引入的；没有 STP，环路会泛洪。
```

规则：每道题所有 `correct` 选项都被选中且无错选才算对；全部题通过才推进；答错显示所选与正确选项的 `explanation`，可重试。quiz 题不部署容器。

headless 判分通过 `--answers <file>` 提供，格式为 `exercise FullID -> [每题所选选项索引列表]`，**索引从 0 开始**；例如：
```yaml
"01-concepts/q1":
  - [0]        # 第 1 题选第 1 个选项
  - [1, 2]     # 第 2 题选第 2、3 个选项
```
未出现在文件中的 quiz 视为跳过（`test --all` 默认行为）；出现在文件中但答案列表为空/长度不符则判失败。

### 6.5 阅读切片 `reading.<lang>.yaml`

```yaml
- rfc: 2328
  title: OSPF Version 2
  section: "7.3"
  section_title: The Database Description Packet
  kind: required           # required | optional
  url: https://www.rfc-editor.org/rfc/rfc2328#page-79
  intro: {en: ..., zh: ...}     # 2–3 句导读：这节讲什么、和本实验的关系
  question: {en: ..., zh: ...}  # 一个"带着读"的问题
```

约束：切片只给"指针 + 导读 + 问题"，不复制 RFC 大段原文。

### 6.6 断言 `wait`

- `wait` 为可选时长。设置后，断言在 `wait` 窗口内以固定间隔重复执行，直到匹配通过或窗口耗尽；未设置则单次执行。
- `wait` 是**总预算**（从首次尝试前开始计时）：窗口内以固定间隔重试；睡眠会夹紧到剩余时间并可被 `ctx` 取消；**不会在截止之后启动新的尝试**。单次命令执行仍受断言 `timeout`/题目 `timeout` 约束。

## 7. 语言选择与查看

- 语言解析优先级：`--lang`（`list/watch/run/reset/test/read` 等）+ `CLABLINGS_LANG` 环境变量 + 配置 `language`（`~/.config/clablings/config.yaml`） + 默认 `en`。
- **内容**（章节/题名、hint、concept、reading、quiz）随语言切换，缺失回退英文。
- **CLI 文案**（进度、报告标签、错误信息）恒英文。
- 新增 `clablings read [<chapter-id>] [--lang]`：输出指定/当前章节的 concept 与 reading 切片。
- watch 内新增按键 `c`：查看当前章 concept + reading（按当前语言）。

## 8. 由内容推导的引擎/工具缺口

1. **断言轮询 `wait`**（STP/路由/收敛类必需）。
2. **`quiz` 题型**：无容器验证路径、答题交互（watch 与 headless）、门控推进、可重试。
3. **可选拓扑**：章节/题目可为无 topology（纯概念/quiz）；容器操作（deploy/destroy/redeploy）对其 no-op，但**宿主 setup/teardown 仍会执行**。交互式 quiz 作答见 §6.4 + Plan 3（未接入前，watch 遇到 quiz 会明确报错而非静默卡住）。
4. **多语言解析**：`LocalizedString` + locale 解析与回退，贯穿 content/report/watch/read。
5. **L2 状态断言**：优先用 `exec` + JSON/text 匹配（bridge/bond/fdb/vxlan 状态）；如不足再细化 `interface`。
6. **交换底座镜像**：SR Linux（主）/cEOS/SONiC 的 topology 支持与 preflight 镜像检查（现有从拓扑收集镜像的机制可复用）。
7. **`clablings lint`**：校验 schema、拓扑文件存在、断言的 node/interface 引用、多语言缺翻译、quiz 答案完整性、`wait` 合法性。
8. **元数据**：解析并在 `list` 中展示 difficulty/tags/objectives/prerequisites；quiz 进度纳入状态机。
9. **`make install` / `make uninstall`**：默认安装到 `~/.local/bin`（见 §10）。

## 9. 出题与校准工作流

1. 写 `chapter.yaml` + `concept.<lang>.md` + `reading.<lang>.yaml` + quiz/实验题。
2. `clablings lint --root <library>` 静态校验。
3. **真机校准**：部署 → 确认初始断言**必失败** → 应用正确修复/实现 → 确认断言**必通过** → 还原故障态。
4. 多语言检查：默认英文完整；中文可逐步补齐，缺翻译不得导致解析失败（回退英文）。
5. 提交（题库为独立 git 仓库）。

## 10. 工具链：安装与卸载

`clablings/Makefile`：
- `PREFIX ?= $(HOME)/.local`，即默认安装到 `~/.local/bin`。
- `install`：构建静态二进制并 `install -m 0755` 到 `$(DESTDIR)$(PREFIX)/bin/clablings`。
- `uninstall`：`rm -f $(DESTDIR)$(PREFIX)/bin/clablings`。
- `help` 同步说明；保留 `PREFIX`/`DESTDIR` 覆盖能力（打包/系统安装）。

## 11. 验收标准

- `clablings lint` 能对题库给出结构化校验结果，非法内容有明确错误。
- M1 五章可端到端跑通：每章 `concept → quiz → 实验` 门控推进；STP/收敛类题依赖 `wait` 稳定通过。
- `clablings read` 与 watch `c` 键能按当前语言显示 concept/reading，缺翻译回退英文。
- 内容可 `en`/`zh` 切换，CLI 文案恒英文。
- `make install` 后 `~/.local/bin/clablings` 可用；`make uninstall` 移除之。
- 现有三题按 §4 迁移并补齐 concept/reading/quiz/objectives 与多语言，真机校准通过。

## 12. 明确不做

- 真机交换机堆叠（StackWise/VSS 等私有机制）的模拟——仅讲概念并用 MLAG/EVPN-MH 替代实践。
- PVST+/MSTP 等厂商专有多实例特性的实践。
- Web UI / 图形化报告。
- `en`/`zh` 之外的语言。
- 抓包类（packet）断言——除非某题非它不可，届时单独评估。
