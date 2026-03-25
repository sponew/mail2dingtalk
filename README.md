# mail2dingtalk

邮件到钉钉机器人转发服务。接收 SMTP 邮件，解析后通过钉钉机器人发送通知。

## 功能特性

- ✅ SMTP 服务器监听（默认端口 2525）
- ✅ 解析纯文本/HTML 邮件
- ✅ HTML 自动转 Markdown
- ✅ 附件处理和存储
- ✅ 附件过大提醒（>20MB 可配置）
- ✅ 钉钉机器人 Markdown 消息
- ✅ 邮件持久化存储（JSON 格式）
- ✅ 自动清理旧邮件（默认 180 天）
- ✅ 并发控制（默认 10 个工作协程）
- ✅ 完整的日志记录

## 快速开始

### 1. 编译

```bash
cd mail2dingtalk
go build -o mail2dingtalk
```

### 2. 配置

编辑 `config.yaml`：

```yaml
smtp:
  port: 2525          # SMTP 监听端口
  domain: "localhost"

dingtalk:
  webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
  secret: ""          # 可选，加签密钥

storage:
  email_dir: "data/emails"
  attachment_dir: "tmp/attachments"
  retention_days: 180  # 邮件保留天数

server:
  max_concurrent: 10   # 最大并发处理数

attachment:
  max_size_mb: 20      # 附件大小限制

log:
  level: "info"
  file: "logs/app.log"
```

### 3. 运行

```bash
./mail2dingtalk
```

### 4. 测试

使用 `swaks` 或任何 SMTP 客户端测试：

```bash
# 安装 swaks
apt-get install swaks

# 发送测试邮件
swaks --to test@example.com \
      --from firewall@company.com \
      --server localhost:2525 \
      --subject "测试告警" \
      --body "这是一封测试邮件"
```

或使用 Python 快速测试：

```python
import smtplib
from email.mime.text import MIMEText

msg = MIMEText("测试邮件内容")
msg['Subject'] = '测试主题'
msg['From'] = 'test@example.com'
msg['To'] = 'admin@example.com'

with smtplib.SMTP('localhost', 2525) as server:
    server.send_message(msg)
print("发送成功!")
```

## 目录结构

```
mail2dingtalk/
├── main.go              # 程序入口
├── config.yaml          # 配置文件
├── go.mod
├── config/
│   └── config.go        # 配置加载
├── smtp/
│   └── server.go        # SMTP 服务器
├── parser/
│   └── email.go         # 邮件解析
├── dingtalk/
│   └── client.go        # 钉钉客户端
├── storage/
│   └── email.go         # 邮件存储
├── data/
│   └── emails/          # 邮件记录存储目录
├── tmp/
│   └── attachments/     # 附件存储目录
└── logs/
    └── app.log          # 日志文件
```

## 钉钉机器人配置

### 1. 创建机器人

1. 打开钉钉群 → 群设置 → 智能群助手
2. 添加机器人 → 自定义
3. 复制 Webhook 地址

### 2. 安全设置（可选）

推荐使用"加签密钥"：
1. 复制密钥到 `config.yaml` 的 `dingtalk.secret`
2. 程序会自动处理签名

## 防火墙配置示例

### iptables

```bash
# 允许 2525 端口
iptables -A INPUT -p tcp --dport 2525 -j ACCEPT
```

### firewalld

```bash
firewall-cmd --permanent --add-port=2525/tcp
firewall-cmd --reload
```

## Systemd 服务（推荐生产环境使用）

### 方式一：使用安装脚本（推荐）

```bash
# 安装服务（需要 root 权限）
sudo ./install-service.sh install

# 卸载服务
sudo ./install-service.sh uninstall
```

安装脚本会自动：
- 复制二进制文件到 `/usr/local/bin/mail2dingtalk`
- 复制配置文件到 `/etc/mail2dingtalk/config.yaml`
- 创建 systemd 服务文件
- 创建必要的目录（数据、日志、附件）
- 自动备份已有配置文件

### 方式二：手动安装

创建 `/etc/systemd/system/mail2dingtalk.service`：

```ini
[Unit]
Description=Mail to DingTalk Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/path/to/mail2dingtalk
ExecStart=/path/to/mail2dingtalk/mail2dingtalk
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
systemctl daemon-reload
systemctl enable mail2dingtalk
systemctl start mail2dingtalk
systemctl status mail2dingtalk
```

## 日志查看

```bash
# 实时查看日志
tail -f logs/app.log

# 使用 journalctl（systemd 方式）
journalctl -u mail2dingtalk -f
```

## 注意事项

1. **端口权限**：端口 2525 不需要 root 权限，如果使用 25 端口需要 root 或设置 capabilities
2. **附件存储**：附件保存在 `tmp/attachments/` 目录，文件名格式为 `{邮件ID}_{原文件名}`
3. **并发控制**：超过 `max_concurrent` 的邮件会进入队列等待
4. **清理策略**：每天运行一次清理，删除超过 `retention_days` 的邮件和附件

## 故障排查

### 邮件无法接收

1. 检查端口是否监听：`netstat -tlnp | grep 2525`
2. 检查防火墙规则
3. 查看日志：`tail -f logs/app.log`

### 钉钉消息发送失败

1. 检查 webhook URL 是否正确
2. 检查加签密钥是否配置正确
3. 检查服务器时间是否同步（签名依赖时间）
4. 查看钉钉机器人是否被禁用

### 附件过大

附件超过 `max_size_mb` 时：
- 文件会保存到本地
- 钉钉发送提醒消息
- 邮件记录中保留附件路径
