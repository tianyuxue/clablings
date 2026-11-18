# Clablings 设计文档
> **已被取代（2026-11-18）**：本文档中关于双语内容、`--lang`、`CLABLINGS_LANG`、`config.language`、`{en, zh}` 映射和语言回退的约定，已由 `2026-11-18-chinese-only-design.md` 取代；其余设计作为历史记录保留。

> 日期：2026-09-17
> 状态：已与用户逐节确认
> 依据：`clablings/prd-design.md` v1.0 (MVP)
> 现有资产：`clab-tui/`（已完成的 Go + Bubble Tea TUI 网络模拟器）

## 1. 背景与目标

Clablings 是面向网络工程师、系统开发者与在校学生的「互动式网络实战与协议通关平台」，把 Rustlings 的小步快跑/即时验证哲学与 Containerlab 的真实容器拓扑结合，用交互式 TUI 做故障排查与协议实现练习。

本文档覆盖 Clablings MVP 的完整技术设计：CLI 调度器、题库内容格式、断言引擎、拓扑生命周期与两级重置、C 协议实现题的构建/下发/进程管理、进度状态、Headless/CI 模式，以及两个示例章节、三道题目（OSPF 运行态排错 + 文件型 config-fix + ARP responder）。

## 2. 已锁定的决策

| # | 决策 |
|---|---|
| 1 | `clablings` 不依赖 `clab-tui` 的代码，直接与 `containerlab` 二进制交互；`clab-tui` 仅作为外部调试台由章节 Makefile 唤起 |
| 2 | 范围：完整 MVP + 三道示例题（两个章节），端到端可跑 |
| 3 | 内容分发：官方题库仓，rustlings 模式（二进制不含题目）；`clablings/` 与 `clab-exercises/` 为独立嵌套 git 仓库，各自管理提交与 CI |
| 4 | 断言：声明式 YAML；类型 = `exec` + `file` + `interface`/`link` |
| 5 | 示例：OSPF MTU 运行态排错（FRR）+ 文件型 config-fix（FRR）+ ARP responder（Linux） |
| 6 | C 题：容器内编译，构建镜像与命令由题目声明 |
| 7 | `make debug`：章节自带 Makefile 直调 `clab-tui --file <本章拓扑>` |
| 8 | 触发：`I AM NOT DONE` 标记门控 + Enter 无条件强制重测；通过后自动推进 |
| 9 | 重置：两级（快路径 + `containerlab redeploy` 兜底） |
| 10 | CLI：`watch` / `run` / `reset` / `list` / `test` / `version` |
| 11 | 重置源：`git checkout` 恢复工作树中已跟踪的题目文件 |
| 12 | 语言约定：代码级文案一律英文（注释、错误信息、CLI 文本、日志、结构化报告标签）；题库内容用中文（题目/断言名/hint/README） |

## 3. 范围

### 3.1 包含

- `clablings` 单二进制 CLI（Go，静态构建）
- 题库内容格式与解析（章节/题目/断言）
- containerlab 驱动：deploy / destroy / redeploy / exec / inspect / 收敛等待
- 两级重置
- 声明式断言引擎与结构化报告（英文标签）
- 文件轮询 + Enter 强制触发的 watch 循环
- C 协议实现题的容器内构建、二进制下发与进程重启
- 进度持久化（`.clablings_state.json`）
- Headless `test` 模式与 CI 适配
- 两个示例章节、三道题目（OSPF 运行态 / 文件型 config-fix / ARP responder）及其 Makefile

### 3.2 明确不做（后续迭代）

- 原始抓包类（`packet`）断言：containerlab CLI 无内置抓包；本机版本 `containerlab tools` 仅含 netem/veth/vxlan/snapshot/gotty/sshx/disable-tx-offload，无 capture。MVP 用节点内探针命令（`vtysh`/`arping`/`ping`/`ip route get`）替代
- 宿主本机编译 C（要求用户装 gcc + 静态 libc，违反轻量化 NFR）
- 节点内编译 C（多数镜像无工具链，SR Linux 不可能）
- 题目内嵌进二进制 / 远程拉取题库
- 多语言支持（MVP 仅 C）
- Web 化 / 图形化报告

## 4. 架构与代码组织

### 4.1 模块

新建独立 Go module：`github.com/tianyuxue/clablings`，目录 `clablings/`，单二进制静态构建。

```
clablings/
  go.mod
  cmd/clablings/main.go
  internal/
    cli/       cobra 命令：watch/run/reset/list/test/version
    content/   题库解析：clablings.yaml / chapter.yaml / exercise.yaml + 目录布局
    engine/    容器与 containerlab 驱动：deploy/destroy/redeploy/exec/inspect/等待收敛
    reset/     两级重置（快路径 / redeploy 兜底）
    build/     容器内 C 构建 + 二进制下发 + 进程重启
    assert/    断言引擎（exec/file/interface/link）+ 结果模型
    watch/     文件轮询 + Enter 触发 + 状态机
    state/     .clablings_state.json 原子读写
    report/    终端彩色报告 + JSON 报告
    config/    ~/.config/clablings/config.yaml
  docs/
```

`clablings/` 与 `clab-exercises/` 均为**独立 git 仓库**（嵌套在外层仓库中，外层通过 `.gitignore` 忽略），与 `clab-tui/` 的形态一致。`clablings` 通过 `--root` / `CLABLINGS_ROOT` 指向题库；两者各自管理提交、分支与 CI。所有 `clablings` 与题库的 git 操作分别在各自仓库内执行（`clablings/`、`clab-exercises/`）。

### 4.2 外部依赖

- `containerlab` 二进制（必需）；`docker` / 容器引擎（必需）
- `clab-tui` 二进制（仅 `make debug` 时需要，且由章节 Makefile 直接调用，clablings 不感知）
- 构建镜像（仅 C 题需要，默认官方 `gcc` 镜像）

### 4.3 题库根定位

`watch`/`run`/`reset`/`list`/`test` 从当前目录向上查找含 `clablings.yaml` 的目录作为题库根；支持 `--root DIR` 与 `CLABLINGS_ROOT` 覆盖。

## 5. 题库内容格式

### 5.1 目录布局

```
clab-exercises/                     # 仓库根，与 clablings/ 并列
  clablings.yaml
  chapters/
    01-ospf-mtu/
      chapter.yaml
      topology.clab.yml        # 含 name: clablings-ospf-mtu
      Makefile                 # make debug -> clab-tui --file topology.clab.yml
      exercises/
        01_mtu_mismatch/       # 运行态排错题（无 files，Enter 触发）
          exercise.yaml
          README.md            # 说明修复路径（TUI 内改运行态 + Enter）
    02-arp-responder/
      chapter.yaml
      topology.clab.yml
      Makefile
      exercises/01_arp_responder/
        exercise.yaml
        arp_responder.c
        Makefile
    03-route-fix/
      chapter.yaml
      topology.clab.yml        # 含 name: clablings-route-fix（无 OSPF，故需静态路由）
      Makefile
      exercises/01_route_fix/  # 文件驱动 config-fix
        exercise.yaml
        r1/routes.sh           # 学员编辑文件（git 已跟踪，可 checkout 还原）
```

### 5.2 `clablings.yaml`（题库根清单）

```yaml
version: 1
name: clablings-exercises
chapters:
  - chapters/01-ospf-mtu
  - chapters/02-arp-responder
  - chapters/03-route-fix
```

### 5.3 `chapter.yaml`

```yaml
id: 01-ospf-mtu
title: OSPF MTU 不匹配
topology: topology.clab.yml
deploy:
  timeout: 120s
setup:            # 可选，章节级、宿主机执行的命令列表；deploy/redeploy 成功后执行
  - "docker exec clab-clablings-ospf-mtu-r1 ip link set eth1 mtu 1400"
teardown:         # 可选，章节级；destroy 之前执行
  - "docker exec clab-clablings-ospf-mtu-r1 vtysh -c 'clear ip ospf process'"
```

- 执行时机：`setup` 在首次 deploy 与硬路径 redeploy 成功后执行；`teardown` 在 destroy 之前执行（跨章切换、headless 跑批结束）。watch 正常退出不 destroy、也不执行 teardown，保留 lab 供 `make debug` 继续排错。
- 失败语义：`setup` 失败中止并报错；`teardown` 失败记录告警但不阻塞销毁。
- 章节拓扑 YAML 内声明 `name: clablings-<chapter-id>`；clablings 不使用 `containerlab --name`，以保证 `clab-tui --file` 解析出同名 lab。
- PRD §3.1 的「题目级幂等 teardown 脚本」由两级 reset 机制替代，`exercise.yaml` 不引入题目级 teardown 字段。

### 5.4 `exercise.yaml`（config-fix，文件驱动）

```yaml
id: 01_route_fix
title: 修复缺失的静态路由
type: config-fix
timeout: 30s
files:                                  # 学员编辑文件 -> 节点目标路径
  - source: r1/routes.sh
    node: r1
    dest: /etc/clablings/routes.sh
reload:                                 # 复制后使配置生效的命令
  - node: r1
    cmd: "sh /etc/clablings/routes.sh"
assertions:
  - name: r1 可到达 r2 环回
    type: exec
    layer: data-plane
    node: r1
    cmd: "ping -c 2 -W 2 10.255.0.2"
    match:
      exit_code: 0
    hint: 检查 routes.sh 中的下一跳地址是否为 10.0.1.2。
```

- 文件驱动题是 `files` + `reload` 机制的内容示例：保存宿主文件并移除哨兵后自动触发；`git checkout` 可还原。

### 5.5 `exercise.yaml`（config-fix，运行态排错）

```yaml
id: 01_mtu_mismatch
title: 修复 OSPF 邻居卡在 ExStart
type: config-fix
timeout: 30s
# 无 files/reload：故障由章节 topology 的 exec 注入，修复在运行态完成
assertions:
  - name: OSPF 邻居达到 Full
    type: exec
    layer: control-plane
    node: r1                             # viewpoint：必须在节点内执行
    cmd: 'vtysh -c "show ip ospf neighbor json"'
    match:
      json: "$.neighbors.*.state"
      equals: "Full"
    hint: 检查链路两侧接口 MTU 是否一致，或使用 ip ospf mtu-ignore。
  - name: r1 可 ping 通 r2 环回
    type: exec
    layer: data-plane
    node: r1
    cmd: "ping -c 2 -W 2 10.255.0.2"
    match:
      exit_code: 0
    hint: OSPF 未收敛时路由不可达，先修复邻居。
```

- 该题**不依赖哨兵门控**：修复方式是在 TUI/vtysh 内改运行态（`ip link set eth1 mtu 1500` 或 `ip ospf mtu-ignore`），随后按 Enter 强制验证；`exercise.yaml` 与章节 README 明确说明。

### 5.6 `exercise.yaml`（code-implement）

```yaml
id: 01_arp_responder
title: 实现 ARP responder
type: code-implement
timeout: 30s
files:                                   # 学员编辑的源码（仅宿主，构建后只下发二进制）
  - source: arp_responder.c
  - source: Makefile
build:
  image: gcc:13
  command: "make"
  output: arp_responder                   # 相对题目目录的产物路径
run:
  node: host1
  dest: /opt/clablings/arp_responder
  stop: "pkill -f /opt/clablings/arp_responder || true"
  start: "nohup /opt/clablings/arp_responder --iface eth1 >/var/log/arp.log 2>&1 &"
assertions:
  - name: 目标主机可解析 r1 的 MAC
    type: exec
    layer: data-plane
    node: host1
    cmd: "arping -c 2 -I eth1 10.0.1.1"
    match:
      exit_code: 0
      stdout:
        contains: "reply from"
    hint: 检查 responder 是否按请求的 target IP 回包，以及是否绑定了正确的接口。
```

### 5.7 标记哨兵

- 哨兵串固定为 `I AM NOT DONE`（匹配子串，注释前缀由内容作者决定，兼容 `//`、`#` 等）。
- 被监听文件保存后若仍包含哨兵串 → 不自动触发；删除哨兵保存 → 自动触发。
- Enter 无条件强制触发（无论哨兵是否存在）。

## 6. 断言模型

### 6.1 通用字段

| 字段 | 说明 |
|---|---|
| `name` | 断言名，报告中显示 |
| `type` | `exec` / `file` / `interface` / `link` |
| `layer` | 可选，`control-plane` / `data-plane`，用于报告分类标签 |
| `node` | 执行视角节点；`exec`/`interface`/`link` 必填，强制节点内视角，禁止宿主机直接探测 |
| `timeout` | 可选，默认取题目 `timeout` |
| `hint` | 失败提示 |

### 6.2 各类型

- `exec`
  - `cmd`：在 `node` 容器内执行的命令
  - `match`：`exit_code`；stdout 的 `equals` / `contains` / `regex` / `json{path, equals|contains}`
  - 支持 `timeout`
- `file`
  - 目标：宿主 `source`（相对题目目录）或 `node` + `path`
  - `contains` / `not_contains`；默认断言不含哨兵 `I AM NOT DONE`
- `interface`
  - `interface`，期望 `state: up|down`（数据来自 `containerlab inspect interfaces`）
- `link`
  - `a` / `port_a` / `b` / `port_b`，期望 `up|down`（数据来自 `containerlab inspect` + 拓扑文件连接关系）

### 6.3 结果模型

```go
type Result struct {
    Exercise  string
    Name      string
    Type      string
    Layer     string // control-plane / data-plane / ""
    Viewpoint string // node name
    Expected  string
    Actual    string
    Hint      string
    Passed    bool
    Err       error
}
```

### 6.4 报告格式

终端（默认彩色）与 JSON 两种输出。报告标签为英文；断言名/hint 来自题库内容，可为中文。终端格式：

```
✗ [data-plane] Client -> Server PBR forwarding
  Expected: packet egress via eth1 (next hop 192.168.1.1)
  Actual:   packet egress via eth2 (default gateway)
  Hint:     check the IP rule table priority or route lookup table ID.
```

- 通过：`✓ [<layer>] <断言名>`
- 分类标签直接使用 `layer` 值（`control-plane` / `data-plane`）；为空则省略。

## 7. 生命周期与两级重置

### 7.1 watch 状态机

```
定位题库 / 读状态 -> 确定当前题目
  -> 确保本章拓扑已部署（否则 containerlab deploy 成功后执行章节 setup）
  -> 对当前题执行快路径重置（git checkout + 复制 files + reload）
  -> 等待触发（文件轮询 + Enter）
     ├─ 失败 -> 打印结构化报告，留在当前题，继续等待
     └─ 通过 -> 写状态
                ├─ 同章下一题 -> 快路径重置，热加载
                └─ 跨章       -> 先章节 teardown，再 destroy 旧拓扑
                                 + deploy 新拓扑 + 章节 setup
```

### 7.2 快路径重置

- config-fix：`git checkout -- <files>` → 重新 `docker cp` 到节点 → 执行 `reload`
- code-implement：`git checkout -- <files>` → 容器内重新编译 → 下发二进制 → 重启进程
- 只对「用户按预期做题」的常见情况可靠（仅改动宿主工作树文件）

### 7.3 硬路径重置

- `clablings reset --hard` → 章节 `teardown`（若声明）→ `containerlab redeploy` → 章节 `setup` → 题目 seed/reload
- 当快路径执行失败、或断言检测到拓扑层被破坏时自动升级为硬路径
- 保证绝对可靠（重建 veth/网络命名空间），代价是数秒级耗时

### 7.4 时延指标（修订 NFR）

- 快路径环境清理与热加载：< 1s
- 硬路径 redeploy：数秒级（镜像预热条件下）
- 代码/配置修改到得出反馈：< 2s（除长超时与硬路径场景）

## 8. 触发与监听

- 文件监听：对声明的 `files` 做轮询（250ms，比对 mtime+size）。`debounce`（默认 300ms）是**稳定期**语义：最近一次变更后需静默超过 `debounce` 才视为「就绪」；`ReadyToVerify()` 需配合轮询/可控时钟测试，不即时断言。轮询而非 fsnotify：少一个依赖、天然覆盖编辑器 rename 替换。
- Enter：TTY 下用 `golang.org/x/term` 进 raw/cbreak，捕获回车无条件强制重测；非 TTY（Headless）自动禁用。
- 单次验证互斥，避免重入。
- 进度行示例：`[3/20] 03_arp_responder: In Progress`（`3/20` 为全局题目序号与总数）
- 与 TUI 不抢 stdio：`watch` 独占一个终端；`make debug` 在另一个终端跑 `clab-tui`。

## 9. C 构建 / 下发 / 进程

1. 构建：`docker run --rm -v <ex-dir>:/src -w /src <build.image> sh -lc "<build.command>"`，产物为 `<ex-dir>/<build.output>`（内容作者应使用静态链接以规避 glibc/musl 差异）
2. 下发：`docker cp <bin> <container>:<run.dest>`
3. 重启：`docker exec <container> sh -lc "<run.stop>; <run.start>"`（`start` 必须后台化，例如 `nohup ... &`）
4. 编译失败视作一次失败验证：结构化输出编译器 stderr，不执行断言
5. 构建镜像与命令全部由题目声明；`build.image` 可选，缺省回退到配置的 `build.default_image`。平台不预置、不自维护镜像。默认示例使用官方 `gcc:13` + `-static`
6. 命令在题目目录内执行，构建产物写到题目目录；实现时应以当前用户身份运行（`docker run --user $(id -u):$(id -g)`）避免产物 root 属主

## 10. 状态与 CLI

### 10.1 状态文件

`.clablings_state.json`（题库根目录），原子写（临时文件 + rename）；损坏时隔离到 `.clablings_state.json.bak` 并给出告警：

```json
{
  "version": 1,
  "current": "01-ospf-mtu/01_mtu_mismatch",
  "exercises": {
    "01-ospf-mtu/01_mtu_mismatch": {
      "status": "passed",
      "attempts": 3,
      "updated_at": "2026-09-17T12:00:00Z"
    }
  }
}
```

`status` 取值：`todo` / `in-progress` / `passed`。

### 10.2 命令

| 命令 | 说明 |
|---|---|
| `clablings watch [--root DIR]` | 主循环：定位/热加载题目、监听文件、Enter 强制重测、通过自动推进 |
| `clablings run <exercise_id>` | 自由跳转到指定题目并加载（不受线性解锁限制） |
| `clablings reset [<exercise_id>] [--hard]` | 重置指定题目（默认当前题目）；快路径 / 硬路径 |
| `clablings list [--json]` | 列出章节/题目/进度 |
| `clablings test [<id>\|--all] [--format text\|json]` | Headless 断言，失败非零退出 |
| `clablings version` | 版本信息 |

`exercise_id` 优先使用全限定 `chapter-id/exercise-id`；也接受题目内 id，但该 id 必须在全题库唯一，否则报错要求使用全限定形式。`test` 在 `--format json` 下**聚合全部题目结果后一次性输出**单个 JSON 数组；退出码由 `main` 统一处理（有失败返回 1，前置/运行错误返回 2）。

## 11. Headless / CI

- `test` 全程不依赖 TTY：不启用 Enter/raw，不显示动画，只跑部署 + 断言 + 结构化报告
- 启动预检（docker daemon、containerlab、所需镜像）失败即 fail fast，给出缺失项与修复提示
- 兼容嵌套虚拟化：仅通过 containerlab/Docker，不额外依赖宿主特权（MVP 断言不需要）
- `test --all`：逐章 deploy（成功后章节 setup）、跑完该章题目（题目间用硬路径重置保证干净）、最后章节 teardown + destroy
- `--format json` 聚合为单个 JSON 数组；CI workflow 置于**仓库根** `.github/workflows/`，用 `defaults.run.working-directory: clablings` 指向模块

## 12. 配置

`~/.config/clablings/config.yaml`（XDG）：

```yaml
containerlab:
  bin_path: ""          # 空 = 从 PATH 解析
docker:
  bin_path: ""          # 空 = 从 PATH 解析
watch:
  poll_interval: 250ms
  debounce: 300ms
build:
  default_image: gcc:13
```

- **接线要求**：`clablings` 各命令入口统一调用 `config.Load()`，用 `containerlab.bin_path` / `docker.bin_path`（空则从 PATH 解析）、`build.default_image`、`watch.poll_interval` / `watch.debounce`；读取失败时打印英文告警并回退默认值。不得在命令中硬编码这些值。

## 13. 错误处理与容错

- 非正常退出（Ctrl+C/崩溃）不损坏代码与状态文件：状态原子写；拓扑保留待 `reset`/`run` 恢复
- 所有外部命令带显式 `timeout`：断言路径由每条断言的 `context.WithTimeout` 保证；章节 deploy/destroy/redeploy、build、headless 跑批等非断言路径由调用方统一 `context.WithTimeout`（deploy/redeploy 用 `deploy.timeout`，构建用题目 `timeout`），不得只依赖 containerlab 的 `--timeout` 参数
- 章节 `setup` 失败中止并报错；章节 `teardown` 失败记录英文告警但不阻塞销毁
- 拓扑部署/收敛失败：打印 containerlab stderr，非零退出，不写通过状态
- `git checkout` 失败（非 git 仓库/文件未跟踪）：明确报错，提示改用 `--hard` 或检查题库仓
- 与 TUI 并发：clablings 拥有生命周期；文档说明 TUI 内的 deploy/destroy 操作可能与 watch 冲突，建议只读/排错使用

## 14. 测试策略

### 14.1 单元测试（假 docker/containerlab runner）

- 内容解析：clablings.yaml / chapter.yaml / exercise.yaml，含非法与缺字段用例
- 状态：原子写、损坏恢复、并发更新
- 断言：四种类型与全部 match 形式的通过/失败用例
- 重置：git checkout 命令构造、快路径/硬路径选择逻辑
- 监听：稳定期（debounce）语义、标记门控、Enter 强制触发、互斥（用轮询/可控时钟测试）
- 报告：英文标签格式与聚合 JSON 结构
- 配置：默认值、覆盖、非法值回退

### 14.2 集成测试（`//go:build integration`）

- 真实部署 FRR 章节与 Linux 章节
- 通过/失败断言、快路径与硬路径重置
- ARP responder：容器内构建 → 下发 → 启动 → 断言通过
- `test --all` headless 全流程
- 自清理：测试结束 destroy，检查无残留容器与网络

## 15. 里程碑

1. content + state + config + list
2. engine（containerlab 驱动）+ 生命周期（章节 setup/teardown）+ 两级重置
3. assert（exec/file/interface/link）+ report
4. watch（文件轮询 + Enter + 稳定期门控 + 自动推进）+ run/reset
5. build（C 容器编译 / 下发 / 进程）
6. 两个示例章节、三道题目 + 章节 Makefile + 使用文档
7. `test` headless + CI workflow（仓库根）+ 集成测试

## 16. 风险与未决问题

- FRR 镜像版本与 `show ... json` 字段结构需在实现时以真实输出校准断言
- OSPF MTU 运行态故障的复现细节（哪侧 MTU 设多少、是否需要 `neighbor ...` 等）需以真实收敛行为验证
- 文件型 config-fix（routes.sh）的初始故障与断言需在真实环境校准
- ARP responder 是否需要 `CAP_NET_RAW`（容器默认 root 通常具备）与接口名稳定性需在示例中固定
- `git checkout` 方案在学员已 commit 的情况下无法还原；文档需明确 reset 面向未提交的工作树改动
- 题库位于本仓 `clab-exercises/` 是 MVP 折中；若后续独立成官方题库仓，`--root` / `CLABLINGS_ROOT` 与 CI 路径需迁移
