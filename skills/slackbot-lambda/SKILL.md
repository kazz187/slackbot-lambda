---
name: slackbot-lambda
description: Use this skill when building a Slack Bot on AWS Lambda with Go using the github.com/kazz187/slackbot-lambda library. Provides API reference, usage patterns, and code generation guidance for event handling, interaction handling, Lambda routing, Slack signature verification, SSM secret loading, and DynamoDB locking.
argument-hint: [task description]
allowed-tools: Read, Grep, Glob, Bash(go *), Write, Edit
---

# slackbot-lambda Library Reference

You are helping the user build a Slack Bot on AWS Lambda using the `github.com/kazz187/slackbot-lambda` library (package name: `slackbot`).

## Import

```go
import slackbot "github.com/kazz187/slackbot-lambda"
```

## API Reference

For the complete source, read Go files in `${CLAUDE_SKILL_DIR}/../../`.

### EventHandler — Slack Event の受信・ルーティング

ジェネリクスとリフレクションで型安全なイベントハンドラーを登録する仕組み。

```go
// EventHandler を作成 (slackVerificationToken は SSMLoader)
eh := slackbot.NewEventHandler(verificationTokenSSM)

// ハンドラー登録 — 型パラメータでイベント種別を指定
slackbot.RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppMentionEvent) error {
    // ev.User, ev.Text, ev.Channel などにアクセス可能
    return nil
})

slackbot.RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppHomeOpenedEvent) error {
    return nil
})

slackbot.RegisterHandler(eh, func(ctx context.Context, ev *slackevents.MessageEvent) error {
    return nil
})

// Lambda ハンドラーとして使用
resp, err := eh.HandleEvent(ctx, apiGatewayRequest)
```

**対応イベント型:** `slackevents` パッケージの任意のイベント型をジェネリクスで指定可能。
- `*slackevents.AppMentionEvent`
- `*slackevents.AppHomeOpenedEvent`
- `*slackevents.MessageEvent`
- その他 `slack-go/slack/slackevents` の全イベント型

**注意:** `HandleEvent` は URL Verification チャレンジも自動処理する。

### InteractionHandler — Slack Interaction の受信・ルーティング

ボタンクリック、モーダル送信などの BlockKit Interaction を ActionID でルーティングする。

```go
ih := slackbot.NewInteractionHandler()

// InteractionType と ActionID でハンドラーを登録
ih.EventRoutes.Register(
    slack.InteractionTypeBlockActions,
    "approve_button",
    func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
        // action.ActionID, action.Value でアクション情報を取得
        // callback.User, callback.Channel でコンテキスト情報を取得
        return nil
    },
)

ih.EventRoutes.Register(
    slack.InteractionTypeViewSubmission,
    "submit_form",
    func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
        // callback.View.State.Values でフォーム入力値を取得
        return nil
    },
)

// Lambda ハンドラーとして使用
resp, err := ih.Handle(ctx, apiGatewayRequest)
```

**対応 InteractionType:**
- `slack.InteractionTypeBlockActions` — ボタン、セレクトメニュー等
- `slack.InteractionTypeBlockSuggestion` — 外部データソースのサジェスト
- `slack.InteractionTypeViewSubmission` — モーダル送信
- `slack.InteractionTypeViewClosed` — モーダルクローズ

### LambdaRouter — パスベースのルーティング

API Gateway のパスで Event と Interaction を振り分ける。

```go
router := slackbot.NewRouter(eh.HandleEvent, ih.Handle)

// Lambda エントリポイント
lambda.Start(router.Handle)
```

**ルーティングルール:**
- パスが `event` で終わる → `EventHandler`
- パスが `interaction` で終わる → `InteractionHandler`

### SSMLoader — AWS Systems Manager Parameter Store からのシークレット読み込み

値を遅延読み込みしてキャッシュする。

```go
// 本番用
ssmCli := ssm.NewFromConfig(awsCfg)
loader := slackbot.NewSSMLoader(ssmCli, "/myapp/slack-verification-token")

// 値の取得 (初回は SSM にアクセス、2回目以降はキャッシュ)
value, err := loader.Get(ctx)
value := loader.MustGet(ctx) // panic on error

// テスト用モック
mock := slackbot.NewSSMLoaderMock("key", "test-value")
mock.SetValue("override-value")
```

### SlackVerificationMiddleware — HTTP ミドルウェア

`net/http` 互換の署名検証ミドルウェア。chi 等のルーターでも使用可能。

```go
r.Use(slackbot.SlackVerificationMiddleware(func(r *http.Request) (string, error) {
    return signingSecretSSM.Get(r.Context())
}))
```

**処理フロー:** リトライリクエストのブロック → ボディ読み取り → 署名検証 → ボディ復元 → 次のハンドラーへ

### API ユーティリティ

```go
// ヘッダー変換 (map[string]string → http.Header)
header := slackbot.ConvertHeaders(request.Headers)

// Slack 署名検証
err := slackbot.Verify(header, body, signingSecret)

// リトライリクエストのブロック
err := slackbot.BlockRetryRequest(header)

// レスポンス生成 (x-slack-no-retry: 1 ヘッダー付き)
resp := slackbot.NewResponse(http.StatusOK)
```

### DynamoDBLock — 分散ロック

DynamoDB の条件付き書き込みで排他制御する。

```go
dynamoDBCli := dynamodb.NewFromConfig(awsCfg)
lock := slackbot.NewDynamoDBLock(dynamoDBCli, "my-lock-table")

// ロック取得 (TTL 付き)
err := lock.AcquireLock(ctx, "user-123", 30*time.Second)

// ロック解放
err := lock.ReleaseLock(ctx, "user-123")
```

**DynamoDB テーブルスキーマ:** パーティションキー `UserID` (String), 属性 `ExpiresAt` (Number, Unix timestamp)

### BotError — エラー型

```go
err := slackbot.NewBotError(slackbot.CodeInternal, "failed to process", originalErr)
// エラーコード: CodeInternal, CodeUnauthorized, CodeInvalidArgument
```

### JSON ユーティリティ

```go
jsonStr := slackbot.MustJSONMarshal(value)    // panic on error
parsed := slackbot.MustJSONUnmarshal[MyType](jsonStr) // panic on error
```

## コード生成ガイドライン

$ARGUMENTS

### 実装する際のルール

1. **必ず `slackbot` パッケージのエイリアスで import する:**
   ```go
   import slackbot "github.com/kazz187/slackbot-lambda"
   ```

2. **Event ハンドラーの登録にはジェネリクス関数 `slackbot.RegisterHandler` を使う** — EventHandler のメソッドではなくパッケージレベル関数。

3. **SSMLoader は必ずコンストラクタで作成する** — フィールドを直接設定しない。テスト時は `NewSSMLoaderMock` を使う。

4. **Lambda のエントリポイントは `router.Handle` を `lambda.Start` に渡す** パターンを推奨。

5. **エラーは `BotError` でラップして返す** ことで、呼び出し元でエラーコードによるハンドリングが可能になる。

### 典型的なプロジェクト構成

```
my-slackbot/
├── cmd/
│   └── bot/
│       └── main.go          # Lambda エントリポイント
├── internal/
│   └── handler/
│       ├── event.go          # Event ハンドラー実装
│       └── interaction.go    # Interaction ハンドラー実装
├── go.mod
└── go.sum
```

### main.go のテンプレート

```go
package main

import (
    "context"
    "log"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/ssm"
    slackbot "github.com/kazz187/slackbot-lambda"
)

func main() {
    ctx := context.Background()
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        log.Fatal(err)
    }
    ssmCli := ssm.NewFromConfig(cfg)

    verificationToken := slackbot.NewSSMLoader(ssmCli, "/myapp/slack-verification-token")

    eh := slackbot.NewEventHandler(verificationToken)
    // slackbot.RegisterHandler(eh, ...) でイベントハンドラーを登録

    ih := slackbot.NewInteractionHandler()
    // ih.EventRoutes.Register(...) でインタラクションハンドラーを登録

    router := slackbot.NewRouter(eh.HandleEvent, ih.Handle)
    lambda.Start(router.Handle)
}
```
