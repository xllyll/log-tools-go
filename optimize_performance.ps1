# 日志查询性能优化 - 快速执行脚本

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "日志查询性能优化工具" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 检查Go是否安装
$goVersion = go version
if ($LASTEXITCODE -ne 0) {
    Write-Host "错误: 未检测到Go环境，请先安装Go" -ForegroundColor Red
    exit 1
}

Write-Host "检测到Go环境: $goVersion" -ForegroundColor Green
Write-Host ""

# 步骤1: 为现有数据库添加索引
Write-Host "步骤 1/3: 为现有数据库添加优化索引..." -ForegroundColor Yellow
if (Test-Path ".\logs_v1.db") {
    Write-Host "找到数据库文件: logs_v1.db" -ForegroundColor Green
    Write-Host "正在运行迁移脚本..." -ForegroundColor Yellow
    
    go run migrate_indexes.go
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ 索引创建成功" -ForegroundColor Green
    } else {
        Write-Host "✗ 索引创建失败，请检查错误信息" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "警告: 未找到 logs_v1.db 文件" -ForegroundColor Yellow
    Write-Host "新索引将在应用启动时自动创建" -ForegroundColor Yellow
}

Write-Host ""

# 步骤2: 重新编译应用
Write-Host "步骤 2/3: 重新编译应用..." -ForegroundColor Yellow
go build -o main.exe

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ 编译成功" -ForegroundColor Green
} else {
    Write-Host "✗ 编译失败" -ForegroundColor Red
    exit 1
}

Write-Host ""

# 步骤3: 提示重启应用
Write-Host "步骤 3/3: 重启应用" -ForegroundColor Yellow
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "优化准备完成！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "请按以下步骤操作：" -ForegroundColor White
Write-Host ""
Write-Host "1. 如果应用正在运行，请先停止 (Ctrl+C)" -ForegroundColor White
Write-Host "2. 重新启动应用:" -ForegroundColor White
Write-Host "   .\main.exe" -ForegroundColor Cyan
Write-Host ""
Write-Host "3. 应用启动后，运行性能测试:" -ForegroundColor White
Write-Host "   go run test_performance.go" -ForegroundColor Cyan
Write-Host ""
Write-Host "预期结果: 响应时间从 2s+ 降低到 200ms 以内" -ForegroundColor Green
Write-Host ""
Write-Host "详细说明请查看: PERFORMANCE_OPTIMIZATION.md" -ForegroundColor Yellow
Write-Host ""
