# Changelog

## 2025-12-18

### 一、Docker 部署支持

| 版本 | 内容 |
|------|------|
| - | 发布 Docker 镜像到 Docker Hub (`kourenicz/pfm`) |
| - | 更新 Dockerfile 使用 Go 1.24 |
| - | 修复 docker-compose.yml 配置 |

### 二、GitHub Actions CI 修复

| 问题 | 解决方案 |
|------|----------|
| Linux `hotkey.ModAlt` 未定义 | 改用 `hotkey.Mod1` |
| Linux 构建需要 X11 显示 | 改为 CLI-only 构建 (`-tags nogui`) |
| Hotkey 代码影响 CLI 构建 | 添加 `//go:build !nogui` 标签 |

### 三、Windows 服务优化

| 版本 | 内容 |
|------|------|
| v1.0.6 | 数据目录改为可执行文件目录（便携式部署） |
| v1.0.7 | 添加管理员权限检查 + 中文错误提示 |
| v1.0.8 | 安装服务后自动启动 |

### 四、GUI 改进

| 版本 | 内容 |
|------|------|
| v1.0.9 | 服务状态实时刷新（3-5秒轮询） |

### 五、新增文件

```
internal/daemon/admin_windows.go   # Windows 管理员权限检查
internal/daemon/admin_other.go     # 非 Windows 平台占位
internal/storage/datadir_windows.go # Windows 数据目录
internal/storage/datadir_darwin.go  # macOS 数据目录
internal/storage/datadir_linux.go   # Linux 数据目录
```

### 六、待办事项（未实现）

| 功能 | 平台 | 状态 |
|------|------|------|
| 服务路径优化（复制到固定目录） | Windows | 待定 |
| 自动更新功能 | 全平台 | 待定 |

---

**当前最新版本：v1.0.9**
