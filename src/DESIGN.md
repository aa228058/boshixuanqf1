# TG-Sender 前端设计令牌规范

> 当前实现以 `web/src/styles/tokens.css` 为唯一设计令牌源。本文档描述代码现状和新增 UI 的约束；旧版 Glassmorphism / Cyberpunk 说明已废止。

## 设计原则

- 以业务可读性和操作效率优先，装饰效果克制。
- 正常、护眼、深色三种主题共享相同的尺寸、字体、圆角、阴影和动效尺度；主题只切换语义颜色。
- 组件样式优先使用共享基类，页面只负责布局，不重复定义按钮、卡片、输入框的外观。
- 不在新样式中直接写色值、rgba、阴影、字号、圆角或任意间距；新增值先评估是否应成为令牌。
- 移动端保留足够触控尺寸，并尊重 `prefers-reduced-motion`。

## 主题颜色

| 语义 | Token | 用途 |
|---|---|---|
| 页面背景 | `--t-bg` | 页面底色 |
| 表面层 | `--t-surface-1/2/3` | 控件、卡片、悬停层级 |
| 主文字 | `--t-text-1` | 标题和正文 |
| 次级文字 | `--t-text-2` | 描述和辅助信息 |
| 弱文字 | `--t-text-3` | 占位符、时间戳 |
| 分隔线 | `--t-line`, `--t-line-strong` | 边框和分隔 |
| 品牌色 | `--t-brand`, `--t-brand-hover`, `--t-brand-active` | 主操作和选中状态 |
| 状态色 | `--t-ok`, `--t-warn`, `--t-err` | 成功、警告、错误 |

状态底色使用 `--t-ok-dim`、`--t-warn-dim`、`--t-err-dim`，不要自行拼接透明色。

## 尺寸与排版

- 间距以 4px 为基准：`--t-space-1` 至 `--t-space-16`。
- 字体使用 `--t-font-*` 或兼容字号 `--t-fs-*`。
- 圆角使用 `--t-radius-*`；圆形标签使用 `--t-radius-pill`。
- 阴影使用 `--t-shadow-xs/sm/md/lg`；键盘焦点使用 `--t-shadow-focus`。
- 动效使用 `--t-duration-*` 和 `--t-ease-*`。
- Tailwind 4 可使用 `p-token-4`、`gap-token-2`、`rounded-token-md`、`text-token-sm`、`shadow-token-sm` 等令牌工具类；自定义 CSS 直接引用 `--t-*`。

## 组件层

| 组件 | 统一类 |
|---|---|
| 内容卡片 | `.ui-card` / `.ui-card--lg` |
| 按钮 | `.ui-btn` + `.ui-btn--primary/secondary` |
| 输入框 | `.ui-input` |
| 状态标签 | `.ui-badge` + `.ui-badge--ok/warn/err/info/mute` |
| 浮层玻璃 | 仅用于导航、遮罩、Toast 等短暂浮层：`.glass-float` |

Modal 使用 `--modal-*` 兼容别名，但颜色和尺寸必须映射到 `--t-*` 令牌。TDesign 通过 tokens.css 的语义桥接变量接入主题。

## 主题规则

- `normal`：默认主题。
- `eye`：护眼主题，运行时同时保留 `theme-eye` 与兼容类 `theme-stripe`。
- `dark`：深色主题。
- 主题文件只允许覆盖语义颜色和第三方组件适配，不应重新定义间距、字号、圆角等全局尺度。

## 新增/修改 UI 的检查清单

1. 颜色来自语义令牌，不直接写 hex/rgb/rgba。
2. 间距优先使用 4px 倍数的 spacing token。
3. 字号、圆角、阴影、层级和动效使用对应 token。
4. 优先复用 UI 基类，不在页面复制基础组件样式。
5. 检查 normal / eye / dark 三种主题。
6. 检查窄屏布局、键盘焦点和减少动态效果设置。
