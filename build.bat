@echo off
echo 正在构建日志分析工具...

REM 安装依赖
echo 安装依赖...
go mod tidy

REM 构建exe文件
echo 构建exe文件...
go build -ldflags="-s -w" -o main.exe

if %ERRORLEVEL% EQU 0 (
    echo 构建成功！
    echo 可执行文件: log-analyzer.exe
    echo 运行命令: log-analyzer.exe
) else (
    echo 构建失败！
    pause
)
