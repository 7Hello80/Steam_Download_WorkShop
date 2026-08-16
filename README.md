# Steam 创意工坊壁纸下载工具 — 搭建与构建指南

本文档面向**开发者 / 自部署用户**，介绍如何从零开始搭建开发环境、安装所需工具，并将源码构建为可运行的二进制文件。支持 **Windows / macOS / Linux** 三大平台，请根据你的操作系统选择对应章节。

> 如果只是想直接运行，不需要修改代码，也可以跳过构建步骤，直接使用他人提供的预编译二进制（`steam-download-tool` / `steam-download-tool.exe`），但仍需完成第 8 章的配置。

---

## 目录

1. [项目简介与目录结构](#1-项目简介与目录结构)
2. [环境要求总览](#2-环境要求总览)
   - [2.1 国内网络环境：更换 Go 模块源](#21-国内网络环境更换-go-模块源)
3. [Windows 搭建](#3-windows-搭建)
4. [macOS 搭建](#4-macos-搭建)
5. [Linux 搭建](#5-linux-搭建)
6. [Android（Termux）搭建](#6-androidtermux-搭建)
7. [从源码构建](#7-从源码构建)
8. [运行前配置](#8-运行前配置)
9. [常见问题（FAQ）](#9-常见问题faq)

---

## 1. 项目简介与目录结构

本工具用于下载 Steam 创意工坊（Workshop）壁纸资源并进行格式转换处理，技术栈如下：

| 部分   | 技术                                      |
| ------ | ----------------------------------------- |
| 后端   | Go 1.24（编译时 `CGO_ENABLED=0`，纯静态） |
| 前端   | Vue 3 + Vite 5 + TypeScript + Element Plus |
| 下载引擎 | DepotDownloader（SteamRE 出品的 .NET 8 工具，运行时依赖） |
| 数据库 | SQLite（`modernc.org/sqlite`，纯 Go 实现，无需额外安装） |

前端构建产物（`web/dist`）通过 Go 的 `embed` 机制**直接嵌入后端二进制**，因此最终只需要一个可执行文件 + 一个 `DepotDownloader` + 一份 `config.yaml` 即可运行。

目录结构：

```
steamDownLoadTool/
├── main.go               # 后端入口
├── go.mod / go.sum       # Go 依赖声明
├── internal/             # 后端业务代码（config/handler/service/queue 等）
├── web/                  # Vue 前端源码（构建产物 web/dist 会被嵌入后端）
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── build.sh              # 一键构建脚本（前端 + 后端）
├── config.yaml           # 运行配置文件
├── DepotDownloader       # 下载引擎（需自行下载，见第 8 章）
├── output/               # 下载产物输出目录
├── static/               # 静态资源目录
├── data/                 # SQLite 数据库文件目录
├── tools/reconvert/      # 辅助转换工具（Go 编写，可选）
├── RePKG.exe             # Windows 壁纸打包辅助工具（可选）
└── lib.dex               # Android 端 mpkg 解包辅助工具（可选）
```

---

## 2. 环境要求总览

| 工具            | 最低版本        | 推荐版本       | 用途                     | 是否必需 |
| --------------- | --------------- | -------------- | ------------------------ | -------- |
| Go              | 1.24            | 1.24.x         | 构建后端                 | ✅ 必需 |
| Node.js（含 npm）| 18              | 20 / 22 LTS    | 构建前端                 | ✅ 必需 |
| Git             | 2.x             | 最新版         | 拉取源码                 | ✅ 必需 |
| DepotDownloader | 最新版          | 最新版         | 运行时下载引擎           | ✅ 必需（运行） |
| .NET 8 Runtime  | 8.0             | 8.0.x         | DepotDownloader 运行依赖 | ⚠️ 部分版本需要 |
| VS Code         | —               | 最新版         | 编辑代码 / 配置文件      | 🔷 Windows 建议安装，Linux/macOS 按需可选 |

> **关于 VS Code**：VS Code 是微软出品的免费编辑器，自带集成终端、Git 支持和 Go/Vue 插件生态。
> - **Windows**：**建议安装**——Windows 默认没有趁手的编辑器与类 Unix 终端，使用 VS Code 内置终端可以一站式完成本文档所有命令。
> - **Linux / macOS**：**按需可选**——习惯命令行（vim/nano）或纯服务器部署的用户可以不装；桌面用户建议安装以获得更好的开发体验。
> - **Android（Termux）**：无法安装桌面版 VS Code，可按需安装 code-server 获得浏览器版 VS Code（见 [6.6 节](#66-关于-vs-code可选)）。

### 2.1 国内网络环境：更换 Go 模块源

> 🔷 仅中国大陆用户需要本小节；网络能直连官方源的用户可跳过。

官方 Go 模块代理 `proxy.golang.org` 在国内经常超时或极慢，构建时的 `go mod tidy` 和依赖下载会卡住甚至失败。**安装完 Go 之后，建议立刻把模块源换成国内镜像**。

打开终端（Windows 用 Git Bash）执行：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=off
```

| 命令 | 作用 |
| ---- | ---- |
| `go env -w GOPROXY=https://goproxy.cn,direct` | 通过七牛云 `goproxy.cn` 镜像下载模块；`direct` 表示镜像上没有时回源官方仓库，兼顾可用性与时效 |
| `go env -w GOSUMDB=off` | 关闭模块校验和数据库。官方校验服务 `sum.golang.org` 国内无法访问，其镜像 `sum.golang.google.cn` 在部分网络环境下也不稳定；直接关闭可确保构建不会因校验服务不可达而失败（代价是跳过依赖校验，自用/内网场景可接受） |

验证是否生效：

```bash
go env GOPROXY GOSUMDB
# 应输出：https://goproxy.cn,direct
#         off
```

> 💡 如果你的网络能稳定访问 Google 的国内镜像，也可以把 `GOSUMDB` 设置为 `sum.golang.google.cn` 以保留依赖校验能力；若设置后构建仍报 `checksum mismatch` 或校验服务超时，请改回 `off`。

其他可用的国内镜像（任选其一即可）：

- `https://goproxy.cn,direct`（七牛云，推荐）
- `https://goproxy.io,direct`
- `https://mirrors.aliyun.com/goproxy/,direct`

如需恢复官方源：`go env -w GOPROXY=https://proxy.golang.org,direct` 与 `go env -w GOSUMDB=sum.golang.org`。

> 💡 前端构建依赖 npm，国内访问官方 registry 同样可能很慢，建议顺带更换：
>
> ```bash
> npm config set registry https://registry.npmmirror.com
> npm config get registry   # 验证
> # 恢复官方源：npm config delete registry
> ```

---

## 3. Windows 搭建

### 3.1 安装 Git for Windows

1. 访问 <https://git-scm.com/download/win> 下载安装包，运行安装。
2. 安装过程中保持默认选项即可（**务必勾选**将 Git Bash 加入右键菜单）。
3. 安装完成后，在开始菜单打开 **Git Bash**，验证：

```bash
git --version
```

> 后续所有命令都建议在 **Git Bash** 中执行，因为项目自带的 `build.sh` 构建脚本依赖 bash 环境。Windows 自带的 CMD / PowerShell 无法直接运行它（PowerShell 的手动构建方法见 7.3）。

### 3.2 安装 VS Code（建议）

1. 访问 <https://code.visualstudio.com/> 下载 Windows 安装包并安装。
2. 安装后，推荐安装以下扩展（点击左侧「扩展」图标搜索）：
   - **Go**（作者 golang.go）——Go 代码补全、调试；
   - **Vue - Official**（作者 Vue.volar）——Vue 3 / TypeScript 支持；
   - （可选）**Chinese (Simplified) Language Pack**——中文界面。
3. 打开 VS Code 内置终端（快捷键 `` Ctrl+` ``），它默认支持 Git Bash，可直接执行本文档所有命令。

### 3.3 安装 Go

1. 访问 <https://go.dev/dl/>，下载 Windows 最新版安装包（如 `go1.24.x.windows-amd64.msi`）并安装，一路下一步即可。
2. 重新打开 Git Bash（让环境变量生效），验证：

```bash
go version
# 应输出类似：go version go1.24.4 windows/amd64
```

> 版本必须 ≥ 1.24（`go.mod` 要求 `go 1.24`），旧版本会直接拒绝构建。
>
> 🔷 国内用户请立即执行 [2.1 节](#21-国内网络环境更换-go-模块源)的命令更换 Go 模块源，避免后续构建时下载依赖卡住。

### 3.4 安装 Node.js（含 npm）

1. 访问 <https://nodejs.org/>，下载 **LTS** 版本的 Windows 安装包（`.msi`）并安装。
2. 重新打开 Git Bash，验证：

```bash
node -v
npm -v
```

### 3.5 获取源码

在 Git Bash 中执行：

```bash
git clone <你的仓库地址>
cd steamDownLoadTool
```

如果是通过其他方式拿到的源码压缩包，解压后进入项目根目录即可。

### 3.6 安装可选辅助工具（按需）

- **RePKG.exe**：Windows 壁纸打包工具，仅当需要「打包」功能时使用，项目根目录已附带该文件。
- **Java 运行时**：仅当需要在 Windows 上通过 `lib.dex` 解包 Android 端 mpkg 文件时需要，访问 <https://adoptium.net/> 安装 Temurin JDK 即可。

### 3.7 构建与运行

```bash
# 在项目根目录（Git Bash）执行一键构建脚本
./build.sh

# 构建完成后启动服务
./steam-download-tool.exe
```

浏览器访问 <http://localhost:8086> 即可看到界面。**首次运行前请先完成第 8 章的配置**（下载 DepotDownloader、配置 `config.yaml`）。

> 不熟悉 Git Bash？也可以完全在 VS Code 内置终端中操作：打开项目文件夹后，`` Ctrl+` `` 呼出终端，默认即为 Git Bash。

---

## 4. macOS 搭建

### 4.1 安装命令行工具与 Homebrew

macOS 自带的 Git 在首次使用时会自动触发 Xcode 命令行工具（Command Line Tools）安装，也可以手动安装：

```bash
xcode-select --install
```

Homebrew 是 macOS 最常用的包管理器，后续 Go / Node.js 都通过它安装：

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

安装完成后按提示将 Homebrew 加入 PATH（Apple Silicon 机型通常需要执行 `echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile` 并重开终端），验证：

```bash
brew --version
```

### 4.2 安装 Go 与 Node.js

```bash
brew install go node
```

验证版本：

```bash
go version   # 需 ≥ 1.24
node -v      # 需 ≥ 18
npm -v
```

> 如果 Homebrew 安装的 Go 版本低于 1.24，可执行 `brew upgrade go`，或到 <https://go.dev/dl/> 下载官方 `.pkg` 安装包。
>
> 🔷 国内用户请立即执行 [2.1 节](#21-国内网络环境更换-go-模块源)的命令更换 Go 模块源，避免后续构建时下载依赖卡住。

### 4.3 安装 VS Code（可选，按需）

习惯终端 + vim/nano 的用户可以跳过。需要图形化编辑器的话：

```bash
brew install --cask visual-studio-code
```

安装后同样推荐 Go（golang.go）与 Vue - Official 扩展，用 VS Code 打开项目文件夹，内置终端可直接执行后续命令。

### 4.4 获取源码

```bash
git clone <你的仓库地址>
cd steamDownLoadTool
```

### 4.5 构建与运行

```bash
./build.sh          # 一键构建
./steam-download-tool   # 启动
```

浏览器访问 <http://localhost:8086>。首次运行前请完成第 8 章的配置。

> macOS 首次运行下载的二进制文件时，若弹出「无法验证开发者」提示，执行：
> `xattr -d com.apple.quarantine ./DepotDownloader` 或 `xattr -d com.apple.quarantine ./steam-download-tool`

---

## 5. Linux 搭建

以下以 Ubuntu / Debian 系为例，其他发行版将 `apt` 替换为对应包管理器（`dnf`/`pacman` 等）即可。

### 5.1 安装 Git 与基础工具

```bash
sudo apt update
sudo apt install -y git curl wget unzip build-essential
```

### 5.2 安装 Go（官方二进制，推荐）

Ubuntu 官方源里的 Go 版本通常偏旧（低于 1.24），**建议直接从官网安装**：

```bash
# 下载并解压到 /usr/local（示例版本，请以 go.dev/dl 最新 1.24.x 为准）
wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz

# 写入环境变量（bash 用户把 .bashrc 换成 .zshrc 等对应文件）
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

go version   # 需 ≥ 1.24
```

> 🔷 国内用户请立即执行 [2.1 节](#21-国内网络环境更换-go-模块源)的命令更换 Go 模块源，避免后续构建时下载依赖卡住。

### 5.3 安装 Node.js（含 npm）

推荐使用 nvm 安装，版本管理更方便：

```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
source ~/.bashrc
nvm install --lts
```

也可以使用 NodeSource 官方源直接安装 LTS：

```bash
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt install -y nodejs
```

验证：

```bash
node -v
npm -v
```

### 5.4 安装 VS Code（可选，按需）

- **纯服务器部署**（无桌面环境）：**不需要安装**，直接使用 vim/nano 编辑配置。
- **桌面环境开发**：可下载官方 `.deb` 安装包（<https://code.visualstudio.com/>），或使用发行版商店中的 code / VSCodium 替代品。

### 5.5 获取源码

```bash
git clone <你的仓库地址>
cd steamDownLoadTool
```

### 5.6 构建与运行

```bash
./build.sh             # 一键构建
./steam-download-tool  # 启动
```

浏览器访问 <http://localhost:8086>。首次运行前请完成第 8 章的配置。

### 5.7 （可选）使用 systemd 常驻后台

生产环境建议注册为 systemd 服务：

```ini
# /etc/systemd/system/steam-download-tool.service
[Unit]
Description=Steam Workshop Download Tool
After=network.target

[Service]
WorkingDirectory=/opt/steamDownLoadTool
ExecStart=/opt/steamDownLoadTool/steam-download-tool
Restart=always
RestartSec=5
# 可选：通过环境变量覆盖配置
Environment=PORT=8086

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now steam-download-tool
sudo systemctl status steam-download-tool   # 查看运行状态
journalctl -u steam-download-tool -f        # 实时查看日志
```

---

## 6. Android（Termux）搭建

Termux 是 Android 上的终端模拟器 + Linux 用户空间，**无需 root**。借助它，你可以在手机上完成本项目的源码构建、交叉编译与开发调试。

> ⚠️ 能力边界：本项目运行时依赖 DepotDownloader（.NET 8 工具），目前 **.NET 8 没有官方的 Android / Termux 运行时**。因此手机端可以完成构建、启动服务并浏览管理界面，但**下载引擎无法在手机上运行**。需要完整下载功能时，请把构建产物部署到 Linux / Windows / macOS 服务器。

### 6.1 安装 Termux

1. 通过 **F-Droid** 安装：<https://f-droid.org/packages/com.termux/>（推荐）。**不要使用 Google Play 版本**——Play 商店版本已停止维护，会导致后续命令报错。也可以从 GitHub Releases 下载 APK：<https://github.com/termux/termux-app/releases>。
2. 打开 Termux，授权存储访问（需要在安卓文件管理器中看到项目文件时才需要）：

```bash
termux-setup-storage
```

3. 更新软件包：

```bash
pkg update && pkg upgrade -y
```

> 🔷 国内用户建议先切换 Termux 软件源镜像（清华 TUNA / 中科大 USTC / 北外 BFSU 均可）：

```bash
termux-change-repo
# 用方向键选择 "Mirrors in China" 分组中的镜像，回车确认
```

### 6.2 安装构建工具

```bash
pkg install -y git golang nodejs-lts
```

Termux 默认 shell 就是 bash，且自带 `find` / `sed` 等工具，项目的 `build.sh` 可以直接运行。

验证版本：

```bash
go version
node -v
npm -v
```

> 如果 `go version` 低于 1.24 但 ≥ 1.21，无需手动升级：`go.mod` 中声明了 `toolchain go1.24.4`，Go 会自动下载对应工具链（前提是已按 [2.1 节](#21-国内网络环境更换-go-模块源) 配置国内模块源）。版本低于 1.21 时执行 `pkg upgrade golang` 更新。

> 🔷 国内用户同样需要执行 [2.1 节](#21-国内网络环境更换-go-模块源) 的 Go 模块源与 npm 镜像配置。

### 6.3 获取源码与构建

```bash
cd ~
git clone <你的仓库地址>
cd steamDownLoadTool
./build.sh
```

手机 CPU 性能有限，首次 `npm install` + 前端构建可能耗时数分钟到十几分钟，建议：

- 在 Wi-Fi + 充电状态下进行；
- 安装 Termux:API 应用后执行 `termux-wake-lock` 防止后台进程被系统休眠杀掉。

构建产物为 Android arm64 原生二进制：`steam-download-tool`。

### 6.4 运行与访问

按 [第 8 章](#8-运行前配置) 准备 `config.yaml`（Android 上 DepotDownloader 无法运行，可跳过放置，下载功能不可用）：

```bash
./steam-download-tool
```

启动后直接在手机浏览器访问 <http://localhost:8086> 即可打开管理界面（账号、配置、公告等功能正常）。

### 6.5 交叉编译到其他平台（可选）

在 Termux 上同样可以交叉编译出服务器用的二进制（先安装 openssh 用于传输）：

```bash
pkg install openssh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o steam-download-tool-linux .
scp steam-download-tool-linux user@你的服务器:/opt/steamDownLoadTool/
```

### 6.6 关于 VS Code（可选）

Android 上无法安装桌面版 VS Code。确有需要时，可在 Termux 中安装 code-server，在手机或电脑浏览器中使用完整的 VS Code 界面：

```bash
pkg install tur-repo
pkg install code-server
code-server   # 启动后按提示访问 http://localhost:8080
```

---

## 7. 从源码构建

### 7.1 一键构建（推荐）

项目根目录自带 `build.sh`，**Windows（Git Bash）/ macOS / Linux / Android（Termux）通用**，会自动完成三件事：

1. **构建前端**：在 `web/` 下执行 `npm install`（首次）与 `npx vite build --outDir dist`；
2. **修复 `_` 前缀文件**：Go 的 `embed` 会忽略以 `_` 或 `.` 开头的文件，脚本会把 `dist` 中的 `_` 前缀文件重命名并同步更新引用；
3. **构建后端**：执行 `go mod tidy` 与 `CGO_ENABLED=0 go build -ldflags="-s -w" -o steam-download-tool .`。

```bash
./build.sh
```

构建成功后，项目根目录会生成 `steam-download-tool`（Windows 上为 `steam-download-tool.exe`）。

### 7.2 构建步骤拆解（手动执行）

如果不想用脚本，或者需要集成到自己的 CI 流程，手动执行以下等价命令：

```bash
# ---- 1. 构建前端 ----
cd web
npm install                    # 首次需要，之后可跳过
npx vite build --outDir dist   # 产物输出到 web/dist

# 注意：需将 dist 中 _ 开头的文件重命名，并替换 js/css/html 中的引用
#（build.sh 中对应 find + sed 逻辑，手动操作需自行等价处理）

# ---- 2. 构建后端（回到项目根目录） ----
cd ..
go mod tidy
CGO_ENABLED=0 go build -ldflags="-s -w" -o steam-download-tool .
```

### 7.3 Windows PowerShell 手动构建（无 Git Bash 时）

如果不想安装 Git Bash，可用纯 PowerShell 完成：

```powershell
# 1. 构建前端
cd web
npm install
npx vite build --outDir dist

# 2. 重命名 _ 前缀文件，并替换所有 js/css/html 中的引用
Get-ChildItem dist -Recurse -File | Where-Object { $_.Name -like "_*" } | ForEach-Object {
    $oldName = $_.Name
    $newName = $oldName.TrimStart("_")
    Rename-Item -LiteralPath $_.FullName -NewName $newName
    Get-ChildItem dist -Recurse -File -Include *.js,*.css,*.html | ForEach-Object {
        $content = Get-Content -LiteralPath $_.FullName -Raw
        $content -replace [regex]::Escape($oldName), $newName | Set-Content -LiteralPath $_.FullName -NoNewline
    }
}

# 3. 构建后端
cd ..
go mod tidy
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o steam-download-tool.exe .
```

### 7.4 交叉编译（可选）

由于后端编译时关闭了 CGO、前端已嵌入二进制，可以在任意一个平台（包括 Termux）上交叉编译出其他平台的可执行文件（**无需重复构建前端**，`web/dist` 已嵌入）：

```bash
# 在任意平台执行：
# Windows amd64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o steam-download-tool.exe .

# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o steam-download-tool .

# macOS Apple Silicon / Intel
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o steam-download-tool-mac .
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o steam-download-tool-mac-intel .
```

### 7.5 前端开发模式（仅开发时使用）

日常开发前端时，可以用 Vite 开发服务器热更新：

```bash
cd web
npm install
npm run dev   # 启动于 http://127.0.0.1:3000，API 请求会代理到后端
```

同时另开一个终端运行后端（后端直接 `go run .` 即可，注意需先保证 `web/dist` 存在，否则 embed 编译失败——可以先执行一次完整构建生成 `web/dist`）。

---

## 8. 运行前配置

### 8.1 放置 DepotDownloader

下载引擎**不会随源码构建**，需要自行获取并放到项目根目录：

| 平台    | 获取方式                                                                                                                                         |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Windows | 访问 <https://github.com/SteamRE/DepotDownloader/releases/latest>，下载 `DepotDownloader-win-x64.zip`，解压出的 `DepotDownloader.exe` 放到项目根目录 |
| Linux   | 下载 `DepotDownloader-linux-x64.zip`，解压后放到项目根目录并重命名为 `DepotDownloader`，执行 `chmod +x DepotDownloader`                              |
| macOS   | 方式一：`brew tap steamre/tools && brew install depotdownloader`；方式二：下载 `DepotDownloader-osx-x64.zip`（Intel）或 `DepotDownloader-osx-arm64.zip`（Apple Silicon）解压放入项目根目录 |

> 若运行 DepotDownloader 时报错提示缺少 .NET，请安装 **.NET 8 Runtime**：
> - Windows/macOS：<https://dotnet.microsoft.com/download/dotnet/8.0>
> - Ubuntu：`sudo apt install -y dotnet-runtime-8.0`

### 8.2 配置 config.yaml

编辑项目根目录的 `config.yaml`，关键配置项：

| 配置项                | 说明                                                         |
| --------------------- | ------------------------------------------------------------ |
| `port`                | 服务监听端口，默认 `8086`                                     |
| `aes_key`             | **必填**。Steam 密码的 AES 加密密钥，请改成自己的强随机字符串，留空会导致启动失败 |
| `jwt_secret`          | JWT 签名密钥，请修改默认值                                    |
| `frontend_url`        | 前端访问地址，用于 CORS 与回调跳转                            |
| `depot_downloader_path` | DepotDownloader 的路径，默认 `./DepotDownloader`（Windows 需写成 `./DepotDownloader.exe`） |
| `database_path`       | SQLite 数据库路径，默认 `./data/steam-download.db`            |
| `output_dir` / `static_dir` | 下载产物 / 静态资源目录                                  |
| `github_client_id` / `github_client_secret` | GitHub OAuth 登录（可选）            |
| `smtp_*`              | 邮箱验证码发送配置                                            |
| `max_workers`         | 最大并发下载数                                                |
| `file_ttl_hours`      | 下载文件保留时长（小时）                                      |

所有配置项也支持**环境变量覆盖**（优先级高于 config.yaml）：`PORT`、`JWT_SECRET`、`AES_KEY`、`GITHUB_CLIENT_ID`、`GITHUB_CLIENT_SECRET`、`MAX_WORKERS`、`FILE_TTL_HOURS`、`DATABASE_PATH`、`DEPOT_DOWNLOADER_PATH`。

> ⚠️ 注意：`config.yaml` 中包含密钥类配置（SMTP 密码、OAuth 密钥等），提交代码或对外分享前请务必替换掉真实值。

### 8.3 启动

```bash
# Linux / macOS
./steam-download-tool

# Windows（Git Bash / CMD / PowerShell）
./steam-download-tool.exe
```

看到 `Database initialized` 日志即为启动成功，浏览器访问 <http://localhost:8086>。

---

## 9. 常见问题（FAQ）

### Q1：`go: go.mod requires go >= 1.24` 报错
本机 Go 版本过旧。到 <https://go.dev/dl/> 安装最新版 Go（Linux 参考 5.2 节用官方 tarball 覆盖安装）。

### Q2：构建后端时报 `web/dist` 不存在 / embed 失败
后端构建前**必须先构建前端**。执行 `cd web && npm install && npx vite build --outDir dist`，或直接运行 `./build.sh` 让脚本自动处理。

### Q3：启动日志出现 `WARNING: DepotDownloader not found`
下载引擎缺失或路径不对。按 8.1 节下载对应平台的 DepotDownloader 放到项目根目录，并检查 `config.yaml` 中 `depot_downloader_path` 的路径（Windows 下注意 `.exe` 后缀）。

### Q4：运行 DepotDownloader 提示缺少 .NET 运行时
安装 .NET 8 Runtime（下载地址见 8.1 节），安装后无需重启系统，重新启动本工具即可。

### Q5：`npm install` / `go mod tidy` 下载依赖很慢或超时
多半是访问官方源不通畅。请按 [2.1 节](#21-国内网络环境更换-go-模块源)更换 Go 模块代理与 npm 镜像后重试。

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=off
npm config set registry https://registry.npmmirror.com
```

如果更换后仍然失败，检查是否设置了系统级 HTTP 代理，或换用其他镜像源（如 `goproxy.io`、阿里云镜像）。

### Q6：端口 8086 被占用
修改 `config.yaml` 中的 `port`，或启动时指定环境变量：`PORT=8087 ./steam-download-tool`。

### Q7：Windows 下 PowerShell 提示「无法加载脚本 / 执行策略受限」
改用 Git Bash 运行 `./build.sh`，或在 PowerShell 中执行 `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned` 后重试。

### Q8：macOS 提示「无法打开，因为无法验证开发者」
对下载的二进制执行 `xattr -d com.apple.quarantine <文件路径>` 解除隔离。

### Q9：构建出来的二进制能直接拷到别的机器用吗？
可以。后端是纯静态编译（`CGO_ENABLED=0`）、前端已嵌入，目标机器**同系统同架构**时直接拷贝 `steam-download-tool`（Windows 为 `.exe`）+ `DepotDownloader` + `config.yaml` 即可运行，无需安装 Go / Node.js。跨系统请参考 7.4 节交叉编译。

