# お便りフォームの作成

Google フォームでリスナーからお便り（リクエスト・質問・感想など）を募り、その回答を RSS フィードとして公開することで、vox-radio のコーナーのデータソースとして取り込めます。番組内でお便りを紹介する、という応用的な使い方です。

> **このページの位置づけ**
> Google フォーム側の設定（フォーム作成・Apps Script・デプロイ）は vox-radio の責務の**範囲外**です。本ページは参考情報として、フォームの回答を vox-radio が読めるフィードに変換する一例を紹介します。手順やサンプルコードは Google の仕様変更により動かなくなる可能性があります。

## 全体の流れ

1. Google フォームでお便りフォームを作成する
2. フォームの Apps Script にサンプルコードを貼り付ける
3. フォームの内容に合わせてサンプルコードの定数を調整する
4. Apps Script を web アプリとしてデプロイする
5. デプロイした web アプリの URL を vox-radio のフィードとして登録する

## 1. お便りフォームを作成する

[Google フォーム](https://forms.google.com/)で、お便りを受け付けるフォームを作成します。たとえば次のような質問を用意します。

- **ラジオネーム**（投稿者名として使う）
- **タイトル**（お便りの件名として使う）
- **本文**（お便りの内容）
- その他、紹介したい任意の質問

このうち「ラジオネーム」「タイトル」は手順3でサンプルコードの定数と対応させるので、質問名を控えておいてください（完全一致でなくても、定数で指定した文字列を含めばマッチします）。

## 2. Apps Script にサンプルコードを貼り付ける

フォーム編集画面の右上メニューから **Apps Script** を開き、エディタの内容を次のサンプルコードで置き換えます。

このスクリプトは、フォームの回答を投稿日時の新しい順に RSS 2.0 フィードとして配信します。

```javascript
const FEED_DESCRIPTION = "フィードの説明"; // フィードの説明
const TITLE_QUESTION = "タイトル"; // フィードのtitleとして利用する質問名
const AUTHOR_QUESTION = "ラジオネーム"; // フィードのauthorとして利用する質問名
const MAX_ITEMS = 20;  // 配信する最大件数

function doGet() {
  const form = FormApp.getActiveForm();
  const formId = form.getId();
  const responses = form.getResponses()
    .sort( (a,b) => b.getTimestamp() - a.getTimestamp()) // 投稿日の降順
    .slice(0, MAX_ITEMS);
  const items = responses.map(r => {
    let title = "新しい回答"; // 件名のデフォルト値
    let author = "匿名"; // 著者名のデフォルト値
    let answers = []; // その他の回答

    // 質問の回答を一つずつチェック
    for (let itemResponse of r.getItemResponses()) {
      const questionName = itemResponse.getItem().getTitle(); // 質問名
      const rawAnswer = itemResponse.getResponse(); // 回答（加工前）
      const answer = Array.isArray(rawAnswer) ? rawAnswer.join(', ') : rawAnswer; // 回答（文字列に加工済み）
      if (questionName.indexOf(TITLE_QUESTION) !== -1) {
        title = answer;
      } else if (questionName.indexOf(AUTHOR_QUESTION) !== -1) {
        author = answer;
      } else {
        answers.push(`【${questionName}】\n${answer}`);
      }
    }
      
    return {
      id: `form_${formId}_${r.getId()}`,
      author: author,
      timestamp: r.getTimestamp(),
      title: title,
      body: answers.join('\n\n')
    }
  });

  const feed = createFeed(form.getTitle(), form.getEditUrl(), items);
  return ContentService.createTextOutput(feed).setMimeType(ContentService.MimeType.RSS);
}


/**
 * フィードの要素オブジェクト
 * 
 * @typedef {Object} FeedItem
 * @property {string} id - ID（投稿ごとに一意であれば何でもOK）
 * @property {string} author - 投稿者名
 * @property {Date} timestamp - 投稿日時
 * @property {string} title - タイトル
 * @property {string} body - 本文
 */

/**
 * フィードの組み立て
 * @param {string} title - フィードのタイトル
 * @param {string} url - フィードのURL
 * @param {FeedItem[]} items - フィードの要素
 * @return {string} 組み立てたフィード
 */
function createFeed(title, url, items) {
  // XML用の特殊文字エスケープ関数
  const escapeXml = unsafe => {
    if (!unsafe) return "";
    return unsafe.toString().replace(/[<>&'"]/g, function (c) {
      switch (c) {
        case '<': return '&lt;';
        case '>': return '&gt;';
        case '&': return '&amp;';
        case '\'': return '&apos;';
        case '"': return '&quot;';
      }
    });
  };

  let rss = '<?xml version="1.0" encoding="UTF-8"?>';
  rss += '<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">';
  rss += '<channel>';
  rss += '<title>' + escapeXml(title) + '</title>';
  rss += '<link>' + url + '</link>';
  rss += '<description>' + FEED_DESCRIPTION + '</description>';
  rss += '<language>ja</language>';
  
  for (let item of items) {
    // 日付をRSS標準フォーマット（RFC 822形式）に変換
    const pubDate = Utilities.formatDate(item.timestamp, "GMT+9", "EEE, dd MMM yyyy HH:mm:ss +0900");
    
    rss += '<item>';
    rss += '<title>' + escapeXml(item.title) + '</title>';
    rss += '<author>' + escapeXml(item.author) + '</author>';
    rss += '<description>' + escapeXml(item.body).replace(/\n/g, '<br />') + '</description>';
    rss += '<pubDate>' + pubDate + '</pubDate>';
    rss += '<guid isPermaLink="false">' + item.id + '</guid>';
    rss += '</item>';
  }
  
  rss += '</channel>';
  rss += '</rss>';
  
  return rss;
}
```

## 3. 定数をフォームに合わせて調整する

サンプルコード冒頭の定数を、作成したフォームに合わせて調整します。

| 定数 | 説明 |
|---|---|
| `FEED_DESCRIPTION` | フィードの説明文 |
| `TITLE_QUESTION` | お便りの件名として使う質問名（この文字列を含む質問の回答が `<title>` になる） |
| `AUTHOR_QUESTION` | 投稿者名として使う質問名（この文字列を含む質問の回答が `<author>` になる） |
| `MAX_ITEMS` | 配信する最大件数（投稿日時の新しい順） |

`TITLE_QUESTION` / `AUTHOR_QUESTION` 以外の質問は、すべて本文（`<description>`）に `【質問名】\n回答` の形式でまとめられます。

## 4. web アプリとしてデプロイする

Apps Script エディタの **デプロイ** → **新しいデプロイ** から、種類に **ウェブアプリ** を選んでデプロイします。

- **アクセスできるユーザー** は、vox-radio（フィードを取得する側）が URL を知っていれば認証なしで取得できる設定（例: 「全員」）にします。
- デプロイ後に表示される **ウェブアプリの URL** を控えておきます。この URL が RSS フィードの配信先になります。

> ブラウザでこの URL を開き、RSS（XML）が表示されることを確認しておくと確実です。

## 5. vox-radio のフィードとして登録する

控えた web アプリの URL を、`episode-spec.yaml` のコーナーのデータソース（`source`）に登録します。

```yaml
corners:
  - id: "listener-mail"
    title: "お便りコーナー"
    content: "リスナーから届いたお便りを紹介する"
    cast: { zundamon: "MC", metan: "MC" }
    length_sec: 120
    source:
      - type: feed
        url: "https://script.google.com/macros/s/xxxxxxxx/exec"  # 控えた web アプリの URL
        max_items: 3   # 収集する最大件数（過去に使った記事は除外して確保）
```

これで `episodegen`（番組生成）の収集（`gather`）ステップがフィードを読み込み、お便りが番組に取り込まれます。`source` の設定の詳細は[episode-spec のリファレンス](../../internal/cli/skills/vox-radio/references/episode-spec.md)を参照してください。

## 注意事項

- **Google 側の設定は vox-radio の責務の範囲外**です。フォーム・Apps Script・デプロイの仕様や手順は Google 側の変更で変わる可能性があります。
- **フィードやリスナー投稿の取り扱いは利用者の責任**です。お便りの内容を番組として公開する場合は、投稿者への周知やプライバシーへの配慮を行ってください（フィードの利用規約まわりの考え方は[DISCLAIMER.md](../../DISCLAIMER.md)も参照）。
