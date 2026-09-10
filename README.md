# SkyImage

SkyImage 是一个现代化的图床系统，采用前后端分离架构。

支持在安装向导中一键迁移 **Lsky Pro 开源版** 的数据（详见[从 Lsky Pro 开源版导入](#从-lsky-pro-开源版导入)）。

# 演示站

🔗 演示地址：https://skyimage.demo.123123223.xyz

| 角色 | 邮箱 | 密码 |
| --- | --- | --- |
| 管理员 | demo@example.com | adminpassword |
| 普通用户 | user@example.com | userpassword |

> ⚠️ 演示站数据会定期清理，请勿上传重要文件。

# 安装

### 建议使用docker部署

## docker:
```bash
# 创建 skyimage 文件夹
mkdir skyimage

# 进入 skyimage 文件夹
cd skyimage

# 下载 docker-compose.yml
curl -O https://raw.githubusercontent.com/nxtcorex/skyImage/refs/heads/main/docker-compose.yml

# 下载 .env
curl -o .env https://raw.githubusercontent.com/nxtcorex/skyImage/refs/heads/main/.env.example

# 启动服务
docker-compose up -d
```

> ⚠️ 启动前必须确保 `.env` 文件已存在。若直接 `docker-compose up` 而宿主机没有 `.env`，Docker 会把 `./.env` 挂载为**目录**，导致安装向导 / 数据库迁移无法保存配置（报 `save database config` 错误）。

启动后访问 `http://localhost:8080` 即可进入安装向导页面。

> 🔐 **安装向导密码**：未安装状态下访问安装向导需要验证安装密码，防止部署期间被他人抢占安装。密码通过环境变量 `INSTALL_PASSWORD` 设置；若未设置，服务每次启动（未安装状态下）会自动生成一个 10 位随机密码并打印到启动日志（`docker compose logs` 可查看），安装完成后门禁自动失效。

### 镜像仓库

SkyImage 镜像发布到多个仓库，可按网络环境选择：

| 仓库 | 镜像地址 | 适用场景 |
| --- | --- | --- |
| Docker Hub | `fishcpy/skyimage:latest` | 默认 |
| GHCR | `ghcr.io/nxtcorex/skyimage:latest` | 海外 |
| CNB | `docker.cnb.cool/nxtcorex/skyimage:latest` | **国内** |

使用 CNB 或 GHCR 镜像时，只需替换 `docker-compose.yml` 中的 `image` 字段即可，例如：

```yaml
image: docker.cnb.cool/nxtcorex/skyimage:latest
```

也可以直接拉取：

```bash
# CNB（国内推荐）
docker pull docker.cnb.cool/nxtcorex/skyimage:latest

# GHCR
docker pull ghcr.io/nxtcorex/skyimage:latest
```

### 数据持久化

Docker 部署会挂载以下目录：
- `./storage/data` - 数据库文件目录
- `./storage/uploads` - 上传文件目录
- `./.env` - 配置文件（安装后自动保存数据库配置）

> 注意：数据库类型与连接信息以 `.env` 为准。请勿在 `docker-compose.yml` 的 `environment:` 中设置 `DATABASE_TYPE` 等变量——环境变量优先级高于 `.env`，会在容器重启后覆盖迁移/安装写入的配置，导致"迁移成功但重启后又切回旧数据库"。若目标 MySQL/PostgreSQL 运行在宿主机，容器内请使用 `host.docker.internal` 代替 `127.0.0.1`。

## 二进制部署

前往 [GitHub Releases](https://github.com/nxtcorex/skyImage/releases) 下载对应平台的预编译包。

### 支持平台

| 平台 | 架构 | 文件名格式 |
| --- | --- | --- |
| Linux | x86_64 | `skyimage-*-linux-amd64.tar.gz` |
| Linux | ARM64 | `skyimage-*-linux-arm64.tar.gz` |
| Windows | x86_64 | `skyimage-*-windows-amd64.zip` |
| macOS | Intel | `skyimage-*-darwin-amd64.tar.gz` |
| macOS | Apple Silicon | `skyimage-*-darwin-arm64.tar.gz` |

### 部署步骤

1. 下载对应平台的压缩包并解压
2. 复制 `.env.example` 为 `.env` 并修改配置
3. 运行 `./skyimage`（Linux/macOS）或 `skyimage.exe`（Windows）
4. 访问 `http://localhost:8080` 完成安装向导

## 源码部署

### 前置要求

- Go 1.24+
- Node.js 18+
- pnpm（推荐）或 npm

### 1. 克隆项目

```bash
git clone https://github.com/nxtcorex/skyImage.git
cd skyImage
```

### 2. 安装依赖

```bash
pnpm install
```

### 3. 构建前端

```bash
pnpm build
```

### 4. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 文件，修改以下配置：

```env
HTTP_ADDR=:8080                    # 服务监听地址
STORAGE_PATH=storage/uploads      # 文件存储路径
PUBLIC_BASE_URL=http://your-domain.com  # 公网访问地址
FRONTEND_DIST=dist                # 前端构建产物目录
```

### 5. 启动服务

```bash
go run ./cmd/api
```

启动后访问 `http://localhost:8080` 进入安装向导页面。

## 从 Lsky Pro 开源版导入

安装向导第一步可选择「从 Lsky Pro 开源版导入」，把旧站数据迁移到 SkyImage：

- **支持范围**：Lsky Pro 开源版（V 2.x）。源数据库覆盖其官方支持的全部四种：**MySQL 5.7+ / PostgreSQL 9.6+ / SQL Server 2017+**（填写数据库连接信息）与 **SQLite 3.8.8+**（填写源数据库目录，即 `database.sqlite` 所在文件夹）。
- **导入内容**：用户、角色组、储存策略、相册、图片记录与站点设置；若填写旧站图片目录（原先的 `storage/app/uploads` 文件夹），导入时会按原路径把图片文件拷贝到新站点，图片链接路径保持不变。
- **只读保证**：导入过程对源数据库只做读取（MySQL 连接强制会话只读、SQLite 以只读模式打开），不会修改源数据。
- **密码兼容**：用户密码哈希原样保留，老用户可直接用原密码登录新站。
- **耗时提示**：导入耗时与数据量（尤其是图片数量）成正比，请在导入期间保持页面开启；导入中断或失败后可重新执行，已导入的数据会按原路径覆盖更新。
- **注意**：导入前请先备份源数据库；旧版数据可能无法完全迁移（例如图片文件缺失、个别路径被禁止导入），未成功的项目会在导入结果中明确列出。

## 技术栈

### 后端
- **Go 1.24+** - 高性能后端语言
- **Gin** - Web 框架
- **GORM** - ORM 数据库操作
- **Viper** - 配置管理
- **Cookie + Session** - 身份认证

### 前端
- **React 18** - 用户界面框架
- **TypeScript** - 类型安全
- **Vite** - 构建工具
- **Tailwind CSS** - 样式框架
- **Radix UI** - UI 组件库
- **React Router** - 路由管理
- **Zustand** - 状态管理
- **Axios** - HTTP 客户端

### 数据库支持
- SQLite
- MySQL
- PostgreSQL

安装向导可选择数据库类型；运行中可在 **管理后台 → 系统设置 → 数据库** 将数据迁移到另一种数据库（例如 SQLite → MySQL/PostgreSQL，或反向）。

#### 命令行迁移

```bash
# 将当前 .env 指向的库迁移到 MySQL，并写入 .env 切换运行配置
go run ./cmd/migrate \
  -target-type=mysql \
  -target-host=127.0.0.1 \
  -target-port=3306 \
  -target-name=skyimage \
  -target-user=root \
  -target-password=secret \
  -truncate-target \
  -switch

# 仅测试连接
go run ./cmd/migrate -target-type=sqlite -target-path=storage/data/new.db -dry-run
```

迁移会复制全部业务表数据，不移动已上传的文件（文件仍由存储策略管理）。

注意：
- 目标库非空时须加 `-truncate-target`（或管理端开启「清空目标表」），否则会拒绝迁移
- SQLite 路径必须是相对路径，且位于 `storage/` 下
- 同一进程内同时只能运行一次迁移；请在维护窗口操作

### 存储支持

以下支持情况按当前代码中的存储驱动实现统计；AWS S3、阿里云 OSS、腾讯云 COS、七牛云、又拍云通过 S3 兼容存储驱动接入。

| 支持状态 | 存储类型 |
| --- | --- |
| 支持 | 本地存储、AWS S3、阿里云 OSS、腾讯云 COS、七牛云、又拍云、WebDAV、MinIO |
| 未测试 | FTP、SFTP |

## 主要功能

- 用户注册与登录
- 图片上传与管理
- 存储策略配置
- 用户组与权限管理
- 管理员后台
- 容量监控
- API 文档
- 系统安装向导
- 数据库跨库迁移（SQLite / MySQL / PostgreSQL）
- Turnstile 验证码集成
- 邮件通知

## 项目结构

```
skyimage/
├── cmd/                    # 命令行入口
│   ├── api/               # API 服务
│   └── migrate/           # 跨库数据迁移工具
├── internal/              # 内部包
│   ├── admin/            # 管理员服务
│   ├── api/              # API 处理器
│   ├── config/           # 配置管理
│   ├── data/             # 数据库模型与连接
│   ├── dbmigrate/        # 跨库迁移实现
│   ├── files/            # 文件服务
│   ├── installer/        # 安装服务
│   ├── legacy/           # 旧版导入
│   ├── mail/             # 邮件服务
│   ├── middleware/       # 中间件
│   ├── users/            # 用户服务
│   └── version/          # 版本信息
├── src/                   # 前端源码
│   ├── components/       # React 组件
│   ├── features/         # 功能页面
│   ├── layouts/          # 布局组件
│   ├── lib/              # 工具库
│   └── state/            # 状态管理
└── storage/               # 存储目录
```

## 设计理念

SkyImage 提供了简洁、现代的用户界面和流畅的用户体验。主要设计特点包括：

- 响应式设计，支持多端访问
- 深色/浅色主题切换
- 直观的文件管理界面
- 完善的权限控制系统
- 高效的图片上传体验

## 致谢

[Lsky Pro](https://github.com/lsky-org/lsky-pro) 参考了Lsky Pro的布局和部分逻辑

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=nxtcorex/skyImage&type=date&legend=top-left)](https://www.star-history.com/#nxtcorex/skyImage&type=date&legend=top-left)

## 许可证

本项目（`main` 分支及后续开发）采用 [GNU Affero General Public License v3.0 (AGPL-3.0)](./LICENSE)。

历史代码说明：`skyimage-MIT` 分支中的旧代码仍按 **MIT 许可证** 开源，可单独使用该分支获取 MIT 许可版本。

