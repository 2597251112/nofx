# GitHub Actions 自动部署指南

本文档说明如何为 NOFX 项目配置 GitHub Actions 自动部署到阿里云服务器。

## 部署架构

```
GitHub (Push) 
    ↓
GitHub Actions Workflow 
    ↓
SSH 连接到阿里云服务器
    ↓
Docker Compose 拉取镜像并重启服务
    ↓
应用自动更新（http://YOUR_SERVER_IP:3000）
```

## 配置完成

已配置 4 个 GitHub Secrets：
- `ALIYUN_SSH_PRIVATE_KEY`: SSH 私钥
- `ALIYUN_HOST`: 47.79.252.157
- `ALIYUN_USER`: root
- `ALIYUN_DEPLOY_DIR`: /opt/nofx

## 触发方式

- Push 到 main/dev 分支自动触发
- GitHub Actions 页面手动触发

## 验证部署

SSH 连接服务器检查：
```bash
ssh root@47.79.252.157
docker compose -f /opt/nofx/docker-compose.prod.yml ps
```

或访问：http://47.79.252.157:3000
