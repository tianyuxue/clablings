# 安全策略

## 支持的版本

目前只对最新发布版本提供安全修复。

## 漏洞报告

请不要通过公开 GitHub issue 报告安全漏洞。

请使用本仓库的 GitHub 私密漏洞报告功能：
<https://github.com/tianyuxue/clablings/security/advisories/new>

报告请包含漏洞描述、复现步骤、受影响版本和影响范围。我们会尽快确认收到报告，
并协调修复与披露安排。

## 范围说明

clablings 是本地 CLI。它会调用 `containerlab` 和 Docker CLI，在题库旁生成派生拓扑，
并在实验节点上运行练习命令和断言。提权、命令注入、生成文件的不安全处理，以及
`sudo`/PATH 行为问题都属于报告范围。containerlab、Docker 或 Linux 内核自身的漏洞
请向上游项目报告。
