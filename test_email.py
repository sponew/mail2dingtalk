#!/usr/bin/env python3
"""
邮件测试脚本 - 用于测试 mail2dingtalk 服务
"""

import smtplib
import sys
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from email.mime.base import MIMEBase
from email import encoders

SMTP_SERVER = "localhost"
SMTP_PORT = 2525
FROM_EMAIL = "firewall@example.com"
TO_EMAIL = "admin@example.com"

def send_plain_text():
    """发送纯文本邮件"""
    print("发送纯文本测试邮件...")
    
    msg = MIMEText("这是一封纯文本测试邮件\n\n防火墙告警测试", "plain", "utf-8")
    msg["Subject"] = "【测试】纯文本告警"
    msg["From"] = FROM_EMAIL
    msg["To"] = TO_EMAIL
    
    with smtplib.SMTP(SMTP_SERVER, SMTP_PORT) as server:
        server.send_message(msg)
    print("✓ 发送成功\n")

def send_html():
    """发送 HTML 邮件"""
    print("发送 HTML 测试邮件...")
    
    html = """
    <html>
    <body>
        <h2>防火墙告警</h2>
        <table border="1" style="border-collapse: collapse;">
            <tr><th>时间</th><td>2024-01-15 10:30:00</td></tr>
            <tr><th>源 IP</th><td>192.168.1.100</td></tr>
            <tr><th>目标 IP</th><td>10.0.0.1</td></tr>
            <tr><th>动作</th><td style="color: red;">拦截</td></tr>
            <tr><th>规则</th><td>BLOCK_SUSPICIOUS</td></tr>
        </table>
        <p>请尽快处理!</p>
    </body>
    </html>
    """
    
    msg = MIMEText(html, "html", "utf-8")
    msg["Subject"] = "【测试】HTML 格式告警"
    msg["From"] = FROM_EMAIL
    msg["To"] = TO_EMAIL
    
    with smtplib.SMTP(SMTP_SERVER, SMTP_PORT) as server:
        server.send_message(msg)
    print("✓ 发送成功\n")

def send_with_attachment():
    """发送带附件的邮件"""
    print("发送带附件的测试邮件...")
    
    msg = MIMEMultipart()
    msg["Subject"] = "【测试】带附件告警"
    msg["From"] = FROM_EMAIL
    msg["To"] = TO_EMAIL
    
    body = """
    这是一封带附件的测试邮件。
    
    附件包含日志文件。
    """
    msg.attach(MIMEText(body, "plain", "utf-8"))
    
    # 创建测试附件
    test_content = "这是测试附件内容\n" * 100
    attachment = MIMEBase("application", "octet-stream")
    attachment.set_payload(test_content.encode("utf-8"))
    encoders.encode_base64(attachment)
    attachment.add_header(
        "Content-Disposition",
        "attachment",
        filename="test_log.txt"
    )
    msg.attach(attachment)
    
    with smtplib.SMTP(SMTP_SERVER, SMTP_PORT) as server:
        server.send_message(msg)
    print("✓ 发送成功\n")

def send_multipart():
    """发送多部分邮件（HTML + 纯文本）"""
    print("发送多部分测试邮件...")
    
    msg = MIMEMultipart("alternative")
    msg["Subject"] = "【测试】多部分邮件"
    msg["From"] = FROM_EMAIL
    msg["To"] = TO_EMAIL
    
    text = "纯文本版本：这是一封测试邮件"
    html = """
    <html>
    <body>
        <h3>HTML 版本</h3>
        <p>这是一封<strong>测试邮件</strong></p>
        <ul>
            <li>项目 1</li>
            <li>项目 2</li>
        </ul>
    </body>
    </html>
    """
    
    msg.attach(MIMEText(text, "plain", "utf-8"))
    msg.attach(MIMEText(html, "html", "utf-8"))
    
    with smtplib.SMTP(SMTP_SERVER, SMTP_PORT) as server:
        server.send_message(msg)
    print("✓ 发送成功\n")

def main():
    print("=" * 50)
    print("mail2dingtalk 测试脚本")
    print("=" * 50)
    print()
    
    try:
        send_plain_text()
        send_html()
        send_with_attachment()
        send_multipart()
        
        print("=" * 50)
        print("✓ 所有测试邮件发送成功!")
        print("请检查钉钉群消息和 logs/app.log")
        print("=" * 50)
    except Exception as e:
        print(f"✗ 发送失败：{e}")
        print("\n请确保 mail2dingtalk 服务已启动")
        sys.exit(1)

if __name__ == "__main__":
    main()
