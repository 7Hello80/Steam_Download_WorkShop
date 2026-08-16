#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# 检查 git 安装
command -v git &>/dev/null || error "Git 未安装，请先安装 Git。"

# 判断是否为 Git 仓库
if git rev-parse --is-inside-work-tree &>/dev/null; then
    info "当前目录已是 Git 仓库。"
else
    warn "当前目录不是 Git 仓库，正在初始化..."
    git init || error "初始化失败。"
    info "Git 仓库初始化完成。"
fi

# 检查并配置远程仓库 origin
if git remote get-url origin &>/dev/null; then
    info "远程仓库 origin 已存在：$(git remote get-url origin)"
    read -p "是否更换 origin 地址？(y/n，默认 n)：" change_origin
    if [[ "$change_origin" =~ ^[Yy]$ ]]; then
        read -p "请输入新的远程仓库 URL： " new_url
        git remote set-url origin "$new_url" || error "更换 origin 失败。"
        info "origin 已更新为：$new_url"
    fi
else
    warn "未找到远程仓库 origin，请添加。"
    read -p "请输入远程仓库 URL（如 https://github.com/user/repo.git）： " remote_url
    while [[ -z "$remote_url" ]]; do
        read -p "远程仓库 URL 不能为空，请重新输入： " remote_url
    done
    git remote add origin "$remote_url" || error "添加远程仓库失败。"
    info "远程仓库已添加：$remote_url"
fi

# 添加所有更改
info "正在添加所有更改..."
git add . || error "git add 失败。"

# 提交
read -p "请输入提交信息（commit message）： " commit_msg
if [[ -z "$commit_msg" ]]; then
    commit_msg="$(date +'%Y-%m-%d %H:%M:%S') 自动提交"
    warn "提交信息为空，已使用默认信息：$commit_msg"
fi
git commit -m "$commit_msg" || error "提交失败。"

# 分支处理
current_branch=$(git symbolic-ref --short HEAD 2>/dev/null)
if [[ -z "$current_branch" ]]; then
    current_branch="main"
    warn "当前处于 detached HEAD 状态，将使用分支名：$current_branch"
fi

read -p "当前分支为 '$current_branch'，是否使用此分支推送？(y/n，默认 y)： " use_current
use_current=${use_current:-y}
if [[ "$use_current" =~ ^[Yy]$ ]]; then
    branch=$current_branch
else
    read -p "请输入要推送的分支名： " branch
    while [[ -z "$branch" ]]; do
        read -p "分支名不能为空，请重新输入： " branch
    done
    if ! git rev-parse --verify "$branch" &>/dev/null; then
        info "分支 '$branch' 不存在，正在创建并切换..."
        git checkout -b "$branch" || error "创建分支失败。"
    else
        info "切换到分支 '$branch'..."
        git checkout "$branch" || error "切换分支失败。"
    fi
fi

# 推送
info "正在推送到远程仓库 origin/$branch ..."
git push -u origin "$branch" || error "推送失败。请检查网络或权限。"

info "推送成功！"