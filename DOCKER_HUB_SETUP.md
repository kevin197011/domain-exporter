# Docker Hub 设置指南

## 切换到 Docker Hub 的原因
GitHub Packages 权限问题持续存在，Docker Hub 更稳定可靠。

## 设置步骤

### 1. 创建 Docker Hub 账户
1. 访问 [Docker Hub](https://hub.docker.com/)
2. 注册账户或登录现有账户

### 2. 创建访问令牌
1. 登录 Docker Hub
2. 点击右上角头像 → **Account Settings**
3. 选择 **Security** 标签页
4. 点击 **New Access Token**
5. 输入描述（如：GitHub Actions）
6. 选择权限：**Read, Write, Delete**
7. 点击 **Generate**
8. **复制生成的令牌**（只显示一次！）

### 3. 添加到 GitHub Secrets
在你的 GitHub 仓库中：

1. 进入 **Settings** → **Secrets and variables** → **Actions**
2. 添加以下两个 secrets：

   **DOCKERHUB_USERNAME**
   - Name: `DOCKERHUB_USERNAME`
   - Secret: 你的 Docker Hub 用户名（kevin197011）

   **DOCKERHUB_TOKEN**
   - Name: `DOCKERHUB_TOKEN`
   - Secret: 刚才复制的访问令牌

### 4. 验证设置
1. 推送代码到 main 分支
2. 检查 GitHub Actions 是否成功运行
3. 在 Docker Hub 查看是否有新的镜像

## 使用镜像

### 拉取镜像
```bash
docker pull kevin197011/domain-exporter:latest
```

### 运行容器
```bash
docker run -d \
  --name domain-exporter \
  -p 8080:8080 \
  -e DOMAINS="example.com,test.com" \
  kevin197011/domain-exporter:latest
```

### 使用 docker-compose
```bash
docker-compose up -d
```

## 镜像标签说明
- `latest` - 最新的 main 分支构建
- `main` - main 分支的最新提交
- `v1.0.0` - 版本标签（如果有）

## 优势
- ✅ 无权限问题
- ✅ 更快的推送速度
- ✅ 更好的兼容性
- ✅ 免费的公共仓库
- ✅ 支持多平台镜像（amd64, arm64）