# Primind Notification Invoker

Firebase Cloud Messagingに通知送信を要求するワーカー

## エンドポイント

| メソッド | エンドポイント | 概要 |
|---------|------|------|
| POST | /notify | FCM通知を送信 |
| GET | /health | ヘルスチェック |

## Proto定義

- `proto/notify/v1/notify.proto`
- `proto/common/v1/common.proto`
  - Enum: `TaskType`

## 関連リポジトリ
[KasumiMercury/primind-root](https://github.com/KasumiMercury/primind-root)
