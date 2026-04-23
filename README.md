# Linux 更新管理器

自动监控 GitHub Release，保持服务器程序最新的 Web 管理面板。

## 特性

- 🌐 Web 管理面板，卡片式展示所有更新任务
- 🔄 支持**核心**（可执行文件）和**文件**两种更新类型
- 🧠 语义化版本对比（`v1.2.3` 格式），解析失败退回字符串对比
- 📦 自动识别压缩格式：`.zip` `.tar.gz` `.tar.bz2` `.tar.xz`（需系统装 `xz`）及 ELF 直链
- 🎯 模糊关键词评分，自动匹配最佳 Release Asset
- 🌍 下载失败自动尝试 4 个 GitHub 加速代理
- ⏰ 标准 Cron 表达式定时触发
- 🔧 更新前/后执行自定义 Shell 命令
- 💾 JSON 文件持久化，无数据库依赖
- 📋 每个任务独立日志，面板可查看

## 快速开始

### 下载二进制

从 [Releases](../../releases) 下载对应平台的二进制文件。

```bash
chmod +x linux-updater-linux-amd64
./linux-updater-linux-amd64
# 打开浏览器访问 http://your-ip:9191
```

### 命令行参数

```
./linux-updater [选项]

选项：
  --port int    Web 面板端口 (默认 9191)
  --dir  string 数据目录    (默认 <程序目录>/data)
```

示例：
```bash
./linux-updater --port 8080 --dir /var/lib/updater
```

### 自编译

```bash
git clone https://github.com/yourname/linux-updater
cd linux-updater
go build -ldflags="-s -w" -o linux-updater .
```

## 任务配置说明

| 字段 | 必填 | 说明 |
|------|------|------|
| 任务名称 | ✅ | 显示用名称 |
| 更新类型 | ✅ | `core`=可执行文件（自动找 ELF）/ `file`=原样替换 |
| GitHub 项目地址 | ✅ | `https://github.com/owner/repo` |
| 当前版本 | ❌ | 留空则立即执行一次更新并保存版本 |
| 下载文件关键词 | ✅ | 空格分隔多词，模糊评分匹配最佳文件 |
| 重命名 | ❌ | 留空保持原文件名 |
| 放置路径 | ✅ | 绝对路径目录，不存在自动创建 |
| 更新前执行 | ❌ | Shell 命令，例如 `systemctl stop myapp` |
| 更新后执行 | ❌ | Shell 命令，例如 `systemctl start myapp` |
| 定时任务 | ❌ | 标准 5 段 Cron，留空仅手动触发 |

## systemd 服务

```ini
[Unit]
Description=Linux Updater Panel
After=network.target

[Service]
ExecStart=/opt/linux-updater/linux-updater --port 9191 --dir /opt/linux-updater/data
Restart=always
User=root

[Install]
WantedBy=multi-user.target
```

```bash
sudo cp linux-updater.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now linux-updater
```

## 依赖说明

- 处理 `.tar.xz` 需要系统安装 `xz-utils`：`apt install xz-utils` / `yum install xz`
- 其他格式（zip / tar.gz / tar.bz2 / ELF）使用 Go 标准库，无外部依赖
