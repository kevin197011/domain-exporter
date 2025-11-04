# GitHub Personal Access Token 设置指南

## 问题描述
`GITHUB_TOKEN` 权限不足，无法推送到 GitHub Container Registry。需要创建个人访问令牌 (PAT)。

## 解决步骤

### 1. 创建个人访问令牌 (PAT)

1. 登录 GitHub，进入 **Settings** (右上角头像 → Settings)
2. 点击左侧菜单的 **Developer settings**
3. 选择 **Personal access tokens** → **Tokens (classic)**
4. 点击 **Generate new token** → **Generate new token (classic)**

### 2. 配置令牌权限
选择以下权限：
- ✅ `write:packages` - 上传包到 GitHub Packages
- ✅ `read:packages` - 读取包信息
- ✅ `delete:packages` - 删除包（可选）
- ✅ `repo` - 访问仓库（如果是私有仓库）

### 3. 生成并复制令牌
1. 设置令牌过期时间（建议 90 天或自定义）
2. 点击 **Generate token**
3. **立即复制令牌**（只显示一次！）

### 4. 添加到仓库 Secrets
1. 进入你的仓库 `kevin197011/domain-exporter`
2. 点击 **Settings** → **Secrets and variables** → **Actions**
3. 点击 **New repository secret**
4. Name: `GHCR_TOKEN`
5. Secret: 粘贴刚才复制的令牌
6. 点击 **Add secret**

### 5. 更新工作流文件
令牌添加后，工作流会自动使用新的令牌。

## 验证步骤
1. 推送代码触发工作流
2. 检查 Actions 页面的构建日志
3. 成功后在仓库的 Packages 页面查看镜像

## 备用方案：使用 Docker Hub
如果 GitHub Packages 仍有问题，可以切换到 Docker Hub：

```yaml
env:
  REGISTRY: docker.io
  IMAGE_NAME: kevin197011/domain-exporter
```

然后添加 Docker Hub 的用户名和密码到 Secrets。