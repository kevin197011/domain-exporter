# GitHub Container Registry (GHCR) 设置指南

## 新的工作流配置

我已经根据你提供的可用配置更新了工作流，这个配置应该能正常工作。

## 关键改进

1. **简化的权限设置**：只使用必要的权限
2. **移除了 pull_request 条件**：简化登录逻辑
3. **多种标签策略**：
   - `latest` - 最新构建
   - `{sha}` - 完整 commit SHA
   - `{short_sha}` - 短 commit SHA (6位)

## 需要检查的仓库设置

### 1. 工作流权限
在 GitHub 仓库中：
1. 进入 **Settings** → **Actions** → **General**
2. 在 **Workflow permissions** 部分：
   - 选择 **Read and write permissions**
   - 勾选 **Allow GitHub Actions to create and approve pull requests**

### 2. 包可见性设置
1. 进入仓库 **Settings** → **General**
2. 滚动到 **Features** 部分
3. 确保 **Packages** 已启用

### 3. 首次推送后的包设置
第一次成功推送后：
1. 进入仓库的 **Packages** 标签页
2. 找到 `domain-exporter` 包
3. 点击包名进入包设置
4. 在 **Package settings** 中：
   - 设置包的可见性（Public/Private）
   - 如果需要，添加包的描述

## 使用镜像

### 拉取镜像
```bash
# 最新版本
docker pull ghcr.io/kevin197011/domain-exporter:latest

# 特定 commit
docker pull ghcr.io/kevin197011/domain-exporter:abc123

# 完整 SHA
docker pull ghcr.io/kevin197011/domain-exporter:1234567890abcdef...
```

### 运行容器
```bash
docker run -d \
  --name domain-exporter \
  -p 8080:8080 \
  -e DOMAINS="example.com,test.com" \
  ghcr.io/kevin197011/domain-exporter:latest
```

### 使用 docker-compose
```bash
docker-compose up -d
```

## 验证步骤

1. **推送代码**：
   ```bash
   git add .
   git commit -m "Update to GHCR with simplified config"
   git push origin main
   ```

2. **检查 Actions**：
   - 进入仓库的 **Actions** 标签页
   - 查看工作流是否成功运行

3. **检查包**：
   - 进入仓库的 **Packages** 标签页
   - 确认包已成功创建

## 如果仍然失败

如果这个配置仍然失败，可能的原因：

1. **仓库权限问题**：确保你是仓库所有者
2. **组织设置**：如果是组织仓库，可能需要组织级权限
3. **GitHub 服务问题**：可以检查 GitHub Status

## 备用方案

如果 GHCR 仍有问题，我们可以：
1. 回到 Docker Hub 配置
2. 使用其他容器注册表
3. 只进行本地构建

这个新配置基于你提供的可用格式，应该能解决之前的权限问题。