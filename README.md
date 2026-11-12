# TG Sender · UI/UX Prototype

这是一个独立的纯前端 UI/UX 展示分支，不连接 Go 后端、数据库或真实 Telegram 服务。

## 运行

```bash
cd ui-ux
npm install
npm run dev
```

## 设计范围

- Vue 3 + Vite，纯前端单页原型
- 页面数据全部来自本地 Mock 数据
- 页面交互仅在浏览器本地生效，不会创建真实任务或发送消息
- 支持标准浅色、护眼、深色三种主题
- 支持桌面端与移动端布局预览

## 分支边界

本分支只保留 UI/UX 原型及其说明，不包含产品后端、数据库迁移、API、鉴权、Telegram 协议或生产部署配置。