#!/bin/bash
# gost 版本更新检查脚本

set -e

echo "=== gost 版本更新检查 ==="
echo ""

# 当前版本
echo "当前使用的版本:"
go list -m github.com/go-gost/x github.com/go-gost/core 2>/dev/null || echo "未找到依赖"
echo ""

# 最新版本
echo "最新可用版本:"
go list -m -versions github.com/go-gost/x 2>/dev/null | awk '{print $1": "$NF}'
go list -m -versions github.com/go-gost/core 2>/dev/null | awk '{print $1": "$NF}'
echo ""

# 尝试更新
read -p "是否尝试更新到最新版本? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "更新依赖..."
    go get -u github.com/go-gost/x@latest
    go get -u github.com/go-gost/core@latest
    go mod tidy

    echo ""
    echo "尝试编译..."
    if go build . 2>&1; then
        echo "✅ 编译成功！新版本兼容。"
    else
        echo "❌ 编译失败，需要检查 API 变化并修改代码。"
        echo "可以运行 'git checkout go.mod go.sum' 回退版本。"
    fi
fi
