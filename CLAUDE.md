# Port Forward Manager - Claude Code Memory

## 版本号更新位置

每次更新版本号时，需要修改以下文件：

1. **app.go** - 2处
   - `GetStatus()` 函数中的 `Version: "x.x.x"` (约第342行)
   - `GetStatus()` 函数中的 `Version: "x.x.x"` (约第355行)

2. **internal/cli/cli.go** - 1处
   - `handleVersion()` 函数中的 `fmt.Println("Port Forward Manager vx.x.x")` (约第294行)

3. **internal/ipc/server.go** - 1处
   - `GetStatus()` RPC方法中的 `Version: "x.x.x"` (约第343行)

4. **frontend/src/App.vue** - 1处
   - Footer中的 `<span>Port Forward Manager vx.x.x</span>` (约第94行)

5. **frontend/src/views/Settings.vue** - 1处
   - About部分的 `<el-descriptions-item label="Version">x.x.x</el-descriptions-item>` (约第457行)

### 快速搜索命令
```bash
grep -rn "1.0.15" --include="*.go" --include="*.vue" --include="*.ts"
```

## 项目架构

- **后端**: Go + Wails v2
- **前端**: Vue 3 + Element Plus + TypeScript
- **核心引擎**: gost (go-gost/x)

## 关键修复记录

### v1.0.15 - 端口转发服务监听问题
- **问题**: loader.Load() 会清除所有已注册的服务
- **解决**: 使用 service_parser.ParseService() 直接创建服务，避免影响其他已存在的服务
- **文件**: internal/engine/engine.go
