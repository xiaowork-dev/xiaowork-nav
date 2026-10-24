# 贡献指南

感谢参与 XiaoWork Nav。

## 开发原则

1. **后端优先、API 驱动**：接口变更先改 [`docs/openapi/openapi.yaml`](docs/openapi/openapi.yaml)，再改实现与前端。
2. **严格前后端分离**：前端只通过 `/api/v1` 与后端交互；不得直接访问数据库或依赖后端内部结构。
3. **每阶段交付文档**：完成一个阶段后，必须生成 `docs/deliveries/P<N>-delivery.md` 阶段交付报告，并更新 `docs/PROGRESS.md`。规范详见 [`AGENTS.md`](AGENTS.md)。
4. **重要决策写 ADR**：见 `docs/decisions/`。

## 本地开发

```bash
# 后端
cd backend && go run ./cmd/server

# 前端
cd frontend && npm install && npm run dev
```

## 提交前检查

```bash
cd backend  && go vet ./... && go test ./...
cd frontend && npm run build
```

## 提交规范

- 提交信息使用简洁的祈使句，可用 `feat:` / `fix:` / `docs:` / `chore:` 前缀。
- 禁止提交 `.env`、密钥、真实数据文件。

## 许可证

提交即表示同意以 [MIT](LICENSE) 许可证发布贡献内容。
