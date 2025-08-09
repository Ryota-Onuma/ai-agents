# SOLID 原則の実践

オブジェクト指向設計の 5 つの基本原則を実装に適用する能力。

## Single Responsibility Principle (SRP)

- 各クラス/関数は単一の責任のみを持つ
- 変更理由が複数ある場合は分割を検討
- 粒度は文脈と可読性を考慮して決定

## Open/Closed Principle (OCP)

- 拡張に対して開かれ、修正に対して閉じられた設計
- インターフェース分離とポリモーフィズムを活用
- プラグインアーキテクチャの採用

## Liskov Substitution Principle (LSP)

- 派生クラスは基底クラスと置き換え可能
- 契約プログラミングの実践
- 事前条件の弱化、事後条件の強化を禁止

## Interface Segregation Principle (ISP)

- クライアントが不要なメソッドに依存することを禁止
- 小さく特化したインターフェースを作成
- コンポジションによる機能組み合わせ

## Dependency Inversion Principle (DIP)

- 上位モジュールは下位モジュールに依存しない
- 抽象に依存し、具象に依存しない
