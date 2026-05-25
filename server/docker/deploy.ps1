# Windows 本地构建/推送（在 server/docker 目录执行）
#   .\deploy.ps1
#   .\deploy.ps1 -Command push

param(
    [ValidateSet("all", "build", "push", "save", "up", "down")]
    [string]$Command = "all"
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ServerDir = Resolve-Path (Join-Path $ScriptDir "..")
Set-Location $ScriptDir

$envFile = Join-Path $ScriptDir ".env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^\s*#' -or $_ -notmatch '^\s*(\w+)=(.*)$') { return }
        $name = $Matches[1].Trim()
        $val = $Matches[2].Trim().Trim('"')
        Set-Item -Path "env:$name" -Value $val
    }
}

$ImageName = if ($env:IMAGE_NAME) { $env:IMAGE_NAME } else { "log-tools-server" }
$ImageTag = if ($env:IMAGE_TAG) { $env:IMAGE_TAG } else { "latest" }
$Registry = $env:REGISTRY
$Push = if ($env:PUSH) { $env:PUSH } else { "true" }
$SaveTar = if ($env:SAVE_TAR) { $env:SAVE_TAR } else { "false" }
$RunAfter = if ($env:RUN_AFTER_BUILD) { $env:RUN_AFTER_BUILD } else { "false" }

if ($Registry) {
    $ImageFull = "$($Registry.TrimEnd('/'))/$ImageName"
} else {
    $ImageFull = $ImageName
}
$env:IMAGE_FULL = $ImageFull

function Write-Log($msg) { Write-Host "[deploy] $msg" }

function Ensure-Config {
    New-Item -ItemType Directory -Force -Path config, data/uploads | Out-Null
    if (-not (Test-Path "config/config.yaml")) {
        Copy-Item config.example.yaml config/config.yaml
        Write-Log "已生成 config/config.yaml，请修改后重新部署"
    }
}

function Do-Build {
    Write-Log "构建镜像 ${ImageFull}:${ImageTag}"
    docker build -f "$ScriptDir/Dockerfile" -t "${ImageFull}:${ImageTag}" $ServerDir
}

function Do-Push {
    if (-not $Registry) { Write-Log "未设置 REGISTRY，跳过 push"; return }
    docker push "${ImageFull}:${ImageTag}"
}

function Do-Save {
    $tar = "${ImageName}-${ImageTag}.tar"
    docker save -o $tar "${ImageFull}:${ImageTag}"
    Write-Log "已导出 $tar"
}

switch ($Command) {
    "build" { Do-Build }
    "push" { Do-Build; Do-Push }
    "save" { Do-Build; Do-Save }
    "up" { Do-Build; Ensure-Config; docker compose up -d --build }
    "down" { docker compose down }
    default {
        Do-Build
        if ($Push -eq "true" -and $Registry) { Do-Push }
        if ($SaveTar -eq "true") { Do-Save }
        if ($RunAfter -eq "true") {
            Ensure-Config
            docker compose up -d --build
        } else {
            Write-Log "完成。启动: .\deploy.ps1 -Command up"
        }
    }
}
