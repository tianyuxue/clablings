# Quiz 模式快捷键设计

> 日期：2026-10-31
> 状态：已实现
> 范围：`clablings watch` / `clablings run` 中的交互式 quiz 输入

## 问题

quiz 期间 `h`、`c`、`p`、`n`、`q` 可以使用，但 `l`、`r`、`f`、`x` 会被静默忽略，
而上下文图例仍然展示这些按键。原因有二：输入映射不完整；watch 循环与 quiz 同时消费
stdin，导致按键偶尔被错误丢弃。另一个问题是 quiz 没有容器，却尝试执行 containerlab 重部署。

## 决定

- quiz 活跃时将 watch 循环的输入 channel 设为 nil，只让 quiz 处理 stdin，结束后恢复。
- `h`/`c` 显示提示或概念；`p`/`n` 切换练习；`l` 打开选择器；`r`/`f` 重绘上下文；
  `q` 保留容器退出；`x` 销毁题库容器后退出。
- quiz 的 `Session.ResetCurrent` 立即返回，不部署容器。
- quiz 不显示“正在验证”状态行。

## `l` / `r` / `f` 后的行为

quiz 暂停并重新打印上下文，显示“按 Enter 开始 quiz”；再次按 Enter 从第一题开始，避免
半答状态在重置或跳转后残留。

## 验证

测试覆盖四个控制信号、选择器/跳转、重置暂停、清理退出以及 quiz 不显示验证状态；tmux
中人工验证了 `h`、`l`、`r`、`f`。
