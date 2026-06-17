# Mongo 备份恢复演练记录

复制本模板到 `evidence/backup-restore-YYYYMMDD.md`（`evidence/` 默认 gitignore，仅存本地）。

## 元信息

| 字段 | 值 |
|------|-----|
| 日期 | |
| 执行人 | |
| 环境 | compose / 内网生产 |
| Mongo URI | |
| 数据库名 | taskflow |

## 步骤记录

### 1. 基线数据

- [ ] 已存在至少 1 个 `completed` 任务
- [ ] 任务 ID：________________
- [ ] `curl /tasks/:id` 截图或响应摘要：

### 2. 备份

```bash
docker compose exec -T mongo mongodump \
  --uri='mongodb://127.0.0.1:27017/?replicaSet=rs0' \
  --db=taskflow --archive --gzip > backups/evidence-YYYYMMDD.gz
```

- [ ] 备份文件路径：
- [ ] 文件大小（字节）：
- [ ] 完成时间：

### 3. 破坏数据（验证用）

```bash
docker compose exec -T mongo mongosh taskflow --eval 'db.tasks.deleteMany({})'
```

- [ ] 破坏后 `GET /tasks` 为空或任务缺失：是 / 否

### 4. 恢复

```bash
docker compose exec -T mongo mongorestore \
  --uri='mongodb://127.0.0.1:27017/?replicaSet=rs0' \
  --archive --gzip --drop < backups/evidence-YYYYMMDD.gz
```

- [ ] 恢复命令退出码 0
- [ ] 基线任务可再次 `GET /tasks/:id`
- [ ] 用户可正常登录

## 结论

- [ ] **通过** — 备份可用于灾难恢复
- [ ] **未通过** — 原因：________________

## 备注