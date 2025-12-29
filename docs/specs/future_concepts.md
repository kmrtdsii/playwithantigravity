# GitGym Future Concepts: The Mastery Journey

> [!NOTE]
> このドキュメントは、ユーザーから提供された "Skill Radar" UI を基に、GitGym が目指すべき将来的な機能と体験（World View）を提案するものです。

![Git Skill Radar](file:///home/vscode/.gemini/antigravity/brain/4535988a-a1bc-4514-9130-5320807ec5bf/uploaded_image_1766967622406.png)

## 1. Core Concept: "Visual Mastery"

GitGym の世界観は **「Git の内部状態を透明化し、ゲームのように習得する」** ことです。
Skill Radar は単なるメニューではなく、プレイヤー（ユーザー）の **「冒険の地図」** として機能します。

## 2. Feature Ideas

### A. 🎯 Practice Missions (シナリオモード)
Skill Radar の各セクターをクリックすると、そのコマンドを学ぶための具体的な「ミッション」が発動します。

*   **概要**: 砂場（Sandbox）とは異なる、ゴールのあるパズルモード。
*   **例**:
    *   **Level 1 (Basic)**: "初めてのコミット" - ステージングエリアの概念を理解し、ファイルをコミットする。
    *   **Level 3 (Proficient - Rebase)**: "歴史の改変" - 散らかったコミットログを `rebase -i` で綺麗に整えよ。
    *   **Level 5 (God - Reflog)**: "タイムトラベラー" - `reset --hard` で消してしまったコミットを `reflog` から救出せよ。
*   **報酬**: ミッションクリアで Radar のセクターが「Gold（習得済み）」に輝く。

### B. 🧠 AI Git Coach (リアルタイム・メンター)
右下の領域などに常駐する AI エージェントが、ユーザーの操作を見守ります。

*   **機能**:
    *   **Contextual Hints**: ユーザーがエラー（例: コンフリクト）に直面した瞬間、「あ！コンフリクトだね。`git status` で競合ファイルを見てみよう」とアドバイス。
    *   **Transparency**: 裏で走っている実際の Git コマンドの意味を解説（「今のは `index` を書き換えたよ」）。
    *   **褒める**: 難しい操作（Rebaseなど）に成功すると褒めてくれる。

### C. 🔮 Visual Internals (X-Ray Mode)
通常は見えない `.git` ディレクトリの中身を可視化する「レントゲン」モード。

*   **概要**: コマンドを実行したとき、`.git/objects/` や `.git/refs/` の中で何が起きているかをアニメーション表示。
*   **用途**:
    *   `git add` すると Blob オブジェクトが生成される様子が見える。
    *   `git branch` が単なる「コミットへのポインタ（テキストファイル）」作成であることを視覚的に理解させる。

### D. ⚔️ Multiplayer Dojo (対戦・協力モード)
"GitGym" の名の通り、他のユーザーと技を競う、または協力するモード。

*   **Live Pair Ops**: 1つのリポジトリを2人で同時に操作。
*   **Conflict Challenge**: 片方がわざとコンフリクトを起こし、もう片方がそれを素早く解消するタイムアタック。
*   **God Path Race**: Basic から Advanced までのミッションをどれだけ早くクリアできるか競う。

### E. 🛡️ Safe Playground (Undo Anything)
初心者にとって Git は「壊したら戻せない」恐怖があります。GitGym はそれを払拭します。

*   **Global Undo**: どんな操作（`reset --hard` も含む）も、UI上の「Undo」ボタン一発で巻き戻せる機能。
    *   技術的には、操作ごとに GitGym 独自のスナップショットを取ることで実現。
*   **Fearless Exploration**: 「何をしても壊れない」安心感が、積極的な学習（`rebase` や `filter-branch` への挑戦）を促す。

## 3. Roadmap Proposal

| Phase | Feature | Focus |
| :--- | :--- | :--- |
| **Phase 1** | **Skill Radar UI** | Radarの実装、クリックでのヘルプ/Docs表示 (Current) |
| **Phase 2** | **Mission Mode** | 各スキルに対応した基本的なチュートリアルシナリオの実装 |
| **Phase 3** | **Stats & Tracking** | ユーザーごとの習得度保存、Radarの色変化による達成感 |
| **Phase 4** | **God Level (Internals)** | `.git` 内部構造のビジュアライズ、高難易度ミッション |

---
**Next Step**:
このコンセプトの中で、特に「Practice Missions（ミッションモード）」の実装を優先すると、学習アプリとしての価値が最も高まると考えられます。
