# slackbot-lambda

AWS Lambda 上で Slack Bot を構築するための Go ライブラリです。

API Gateway + Lambda 構成における Slack Event / Interaction の受信・ルーティング・署名検証をサポートします。

## Features

- **Event Handler** — ジェネリクスによる型安全な Slack Event ルーティング
- **Interaction Handler** — BlockActions / ViewSubmission 等の Interaction ルーティング
- **Lambda Router** — パスベースで Event / Interaction を振り分け
- **Slack Signature Verification** — リクエスト署名検証 (Lambda / HTTP ミドルウェア両対応)
- **SSM Parameter Store Loader** — シークレットの遅延読み込み・キャッシュ
- **DynamoDB Lock** — DynamoDB ベースの分散ロック

## Install

```bash
go get github.com/kazz187/slackbot-lambda
```

## Quickstart

### 1. SSM Loader でシークレットを準備する

Slack の Verification Token や Signing Secret を AWS Systems Manager Parameter Store から取得します。

```go
package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	slackbot "github.com/kazz187/slackbot-lambda"
)

func setupSSM(ctx context.Context) (*slackbot.SSMLoader, *slackbot.SSMLoader, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	ssmCli := ssm.NewFromConfig(cfg)

	verificationToken := slackbot.NewSSMLoader(ssmCli, "/slack/verification-token")
	signingSecret := slackbot.NewSSMLoader(ssmCli, "/slack/signing-secret")
	return verificationToken, signingSecret, nil
}
```

### 2. Event Handler を登録する

`RegisterHandler` にジェネリクスでイベント型を指定すると、該当イベントが自動的にルーティングされます。

```go
package main

import (
	"context"
	"fmt"

	"github.com/slack-go/slack/slackevents"
	slackbot "github.com/kazz187/slackbot-lambda"
)

func setupEventHandler(verificationToken *slackbot.SSMLoader) *slackbot.EventHandler {
	eh := slackbot.NewEventHandler(verificationToken)

	// @mention イベントを処理
	slackbot.RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppMentionEvent) error {
		fmt.Printf("Mentioned by %s: %s\n", ev.User, ev.Text)
		return nil
	})

	// App Home を開いたイベントを処理
	slackbot.RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppHomeOpenedEvent) error {
		fmt.Printf("App Home opened by %s\n", ev.User)
		return nil
	})

	return eh
}
```

### 3. Interaction Handler を登録する

ボタンクリックやモーダル送信などの Interaction を ActionID ごとにハンドリングします。

```go
package main

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
	slackbot "github.com/kazz187/slackbot-lambda"
)

func setupInteractionHandler() *slackbot.InteractionHandler {
	ih := slackbot.NewInteractionHandler()

	ih.EventRoutes.Register(slack.InteractionTypeBlockActions, "approve_button", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
		fmt.Printf("Approved by %s\n", callback.User.Name)
		return nil
	})

	ih.EventRoutes.Register(slack.InteractionTypeViewSubmission, "submit_modal", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
		fmt.Printf("Modal submitted by %s\n", callback.User.Name)
		return nil
	})

	return ih
}
```

### 4. Lambda Router で起動する

Event と Interaction のハンドラーを Lambda Router にまとめ、API Gateway からのリクエストをパスで振り分けます。

- `*/event` → EventHandler
- `*/interaction` → InteractionHandler

```go
package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	slackbot "github.com/kazz187/slackbot-lambda"
)

func main() {
	ctx := context.Background()

	verificationToken, _, err := setupSSM(ctx)
	if err != nil {
		log.Fatal(err)
	}

	eh := setupEventHandler(verificationToken)
	ih := setupInteractionHandler()

	router := slackbot.NewRouter(eh.HandleEvent, ih.Handle)
	lambda.Start(router.Handle)
}
```

### 5. HTTP ミドルウェアとして使う (Lambda 以外)

`net/http` や chi などの HTTP サーバーでも署名検証ミドルウェアを利用できます。

```go
package main

import (
	"net/http"

	slackbot "github.com/kazz187/slackbot-lambda"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/slack/event", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := slackbot.SlackVerificationMiddleware(func(r *http.Request) (string, error) {
		return "your-signing-secret", nil
	})(mux)

	http.ListenAndServe(":8080", handler)
}
```

## Claude Code Plugin

このライブラリには [Claude Code](https://claude.com/claude-code) 用のプラグインが同梱されています。インストールすると、Claude Code がこのライブラリの API を理解し、コード生成を支援します。

### インストール

```bash
# マーケットプレースを追加
/plugin marketplace add kazz187/slackbot-lambda

# プラグインをインストール
/plugin install slackbot-lambda@slackbot-lambda
```

### 使い方

インストール後、Claude Code に Slack Bot の実装を依頼すると、このライブラリの API を活用したコードを生成します。

```
/slackbot-lambda AppMentionEvent を処理して返信する Bot を作って
```

## License

[MIT](LICENSE)
