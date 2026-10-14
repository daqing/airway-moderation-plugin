# github.com/daqing/airway-moderation-plugin

一个 [Airway](https://github.com/daqing/airway) 插件：把文本按词表过滤的能力做成
HTTP 服务，供其他服务调用。词表只有一处 —— 一份运营可以直接编辑、可以热加载的 JSON
文件 —— 所有调用方问的是同一个问题，宿主与其客户端不会对"哪些内容需要复查"产生分歧。

匹配用 Aho-Corasick 一次扫描整份词表，返回结果里带上命中的词和它所属的类别，运营看到
的是消息**为什么**被拦，而不只是"被拦了"。

## API

挂载在 `/api/v1/moderation`。响应沿用框架信封 —— `{"code": 0, "data": ...,
"message": ""}` —— 出错时 HTTP 状态仍是 200，靠体里的 `code` 区分，调用方必须读响应体：

| Code | 含义 |
| --- | --- |
| `30001` | 请求无法过滤（body 格式不对，或内容超过 32 KiB）。 |
| `30002` | 共享密钥缺失或不正确。 |
| `30003` | 词表加载失败。 |

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/screen` | body `{"content": "..."}`，返回 `{"flagged": bool, "categories": [...], "words": [...]}`。空内容永不命中；命中的词各报一次，按命中结束的先后排序。 |
| `GET` | `/words` | 当前词表（按类别分组）及其来源。 |
| `POST` | `/reload` | 重新读取词表文件，返回与 `/words` 相同的内容。 |
| `GET` | `/health` | 存活探针，无需密钥：`{status, words, source, version}`，`source` 为 `file` 或 `embedded`。不返回路径，也不返回任何词。 |

```bash
curl -sX POST localhost:1905/api/v1/moderation/screen \
  -H 'Content-Type: application/json' \
  -H "X-Moderation-Secret: $MODERATION_SECRET" \
  -d '{"content":"有人在群里发广告，还提到网络赌博，大家小心"}'
# {"code":0,"data":{"flagged":true,"categories":["illegal"],"words":["网络赌博"]},"message":""}
```

### 鉴权

`/screen`、`/words`、`/reload` 由 `MODERATION_SECRET` 里的共享密钥保护，调用方通过
`X-Moderation-Secret` 头提交。本地开发不设置该变量时接口是开放的，此时插件会在启动
日志里给出提示。`/health` 始终不加保护，容器探针不需要密钥。

## 词表

```json
{
  "categories": {
    "illegal": ["网络赌博"],
    "porn": ["裸聊"],
    "weapons": ["仿真枪"]
  }
}
```

文件路径由 `MODERATION_WORDS_FILE` 指定，默认 `config/moderation_words.json`，相对路径
按宿主进程的工作目录解析。加载时会统一转小写并去掉首尾空白，丢弃重复词和空类别；一个
词同时出现在两个类别时归给字母序靠前的那个，因此一次命中只会报出一个类别。

路径不存在时回退到插件内置词表，这样没跑过 `plugin:install` 的 checkout 也能用；
`GET /health` 和 `GET /words` 都会告诉你当前用的是哪一份。文件**存在但解析不了**会让
宿主启动失败；`/reload` 遇到这种文件则返回 `30003`，正在使用的词表继续生效 —— 运营写错
一个字符不会让过滤失效。

改完文件调一次 `/reload` 即可生效，不用重启：

```bash
$EDITOR config/moderation_words.json
curl -sX POST localhost:1905/api/v1/moderation/reload -H "X-Moderation-Secret: $MODERATION_SECRET"
```

### 词从哪来

内置词表由 [Sensitive-lexicon](https://github.com/konsheng/Sensitive-lexicon)（MIT
许可的关键词库）里的高信号类目生成，共 15,652 条、八个类别：`corruption` 176、
`illegal` 78、`political` 8、`porn` 406、`spam` 46、`url` 14,370、`violent` 159、
`weapons` 409。上游版本、许可、用了哪些文件、丢弃了哪些文件及各自原因都记在
`THIRD_PARTY_NOTICES.md`。它不会自动跟随上游更新：换到新版本意味着把上游 checkout
下来、重跑生成器、然后看 diff。

那份词库是给搜索引擎用的，`兼职`、`自拍`、`起诉`、光秃秃一个 `www` 在那里都是有用的
关键词。这里的每一次命中都要占用人工复查，所以生成器会把日常表达剔除：先机械过滤（短于
两个字的词条、短到单独出现没有意义的纯 ASCII 词条），再对照 `tools/lexicon/exclude.txt`
这份人工排除表 —— 它随本仓库提交，就是留给你改的。运营不认可的某个词，从这份文件里删掉，
或者写进 `tools/lexicon/starter_words.json` 与插件自带的手工词表并列，再重跑生成器即可：

```bash
go run ./tools/lexicon -src /path/to/Sensitive-lexicon/Vocabulary \
  -out install/lib/moderation/default_words.json
go run ./tools/lexicon -src /path/to/Sensitive-lexicon/Vocabulary \
  -out install/host/config/moderation_words.json
```

同一份内容还要拷到宿主项目的 `config/moderation_words.json` —— 真正运行时读的是它。
加 `-check` 可以只比较、不写入，用来确认这三处一致。生成器会报告每个类别保留了多少、
丢弃了什么及原因，以及哪些排除词一条都没匹配上（说明上游已经不再收录它）。

有覆盖不等于判断正确：词表告诉复查的人该看什么，而不是什么就是错的；词表里没有的词仍需
人工留意。命中结果里带着具体的词，复查队列自己会说话，方便持续收紧。

## 在宿主应用中使用

```bash
go get github.com/daqing/airway-moderation-plugin
```

然后在宿主的 `plugins.go` 里加空导入启用：

```go
import (
	_ "github.com/daqing/airway-moderation-plugin"
)
```

并把起始词表拷进宿主项目：

```bash
go run . plugin:install github.com/daqing/airway-moderation-plugin
```

`plugin:install` 会把 `install/host/config/moderation_words.json` 写到宿主项目的
`config/moderation_words.json`，已存在的文件不动，所以升级后再跑一次是安全的。本插件
不注册迁移 —— 它不存任何数据。

不想走 HTTP 的调用方可以直接用接口底下的库 `install/lib/moderation`，它被编译进宿主
二进制：

```go
service, err := moderation.New(moderation.WordsFile())
result := service.Screen(content)
```

## 调用方

[Airway IM 插件](https://github.com/daqing/airway-im-plugin) 收到的每条消息都会调用
`IM_MODERATION_URL` 上的 `POST /screen`，并带同一个 `MODERATION_SECRET`（放在
`X-Moderation-Secret` 头里）。它是**失败关闭**的：本服务答不上话时消息进人工复查队列，
所以服务不可用期间所有消息都会等人处理，而不是被投递到会话里。IM 侧复查队列会展示
`/screen` 报回来的命中词。

## 配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `MODERATION_SECRET` | 未设置 | `/screen`、`/words`、`/reload` 的共享密钥。未设置时接口开放。 |
| `MODERATION_WORDS_FILE` | `config/moderation_words.json` | 过滤所用的词表。文件不存在时回退到内置词表。 |

## 开发

```bash
go get github.com/daqing/airway@latest
go mod tidy
go test ./...
```

`install/lib/moderation` 放词表、匹配器和过滤服务，除标准库外不依赖任何东西；
`install/lib/api/moderation_api` 把它们暴露成 HTTP 接口。其中两个测试会把出厂词表分别
跑过一组日常消息和一组必须命中的消息，某个词开始拦截正常聊天时会以测试失败的形式暴露。
`tools/lexicon` 负责生成词表本身，见 `THIRD_PARTY_NOTICES.md`。

## 安装目录约定

`install/lib/` 下的代码通过空导入编译进宿主二进制，不会被拷贝。`install/host/` 下的
文件由 `plugin:install` 拷贝进宿主项目 —— 这里是那份起始词表 —— `install/ignore/`
永远留在插件仓库里。`install/deps/` 用于给宿主附带配套服务；本插件没有。

完整的插件开发指南见
https://github.com/daqing/airway/blob/main/docs/plugin.md
