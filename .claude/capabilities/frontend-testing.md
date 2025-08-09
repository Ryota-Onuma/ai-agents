# Frontend Testing

フロントエンドに特化したテスト戦略と実装パターン。

## テスト種別

### Unit Testing
- コンポーネント単体テスト（React Testing Library）
- カスタムフック単体テスト（@testing-library/react-hooks）
- ユーティリティ関数テスト（Jest）
- 純粋関数とビジネスロジックの検証

### Integration Testing
- コンポーネント間の連携テスト
- API モックを使用したデータフローテスト
- ルーティングテスト（React Router）
- Context/State 管理ライブラリとの統合テスト

## テストツールと設定

### 推奨ツールスタック
- **Jest**: テストランナーとアサーション
- **React Testing Library**: React コンポーネントテスト
- **MSW (Mock Service Worker)**: API モック
- **Storybook**: ビジュアル回帰テスト

### カバレッジ基準
- Lines: >= 80%
- Branches: >= 70%
- Functions: >= 80%
- Statements: >= 80%

## テストパターン

### コンポーネントテスト
- Props による動作変化の検証
- イベントハンドラーの動作確認
- 条件分岐による表示切り替えテスト
- アクセシビリティテスト（@testing-library/jest-dom）

### API 連携テスト
- 正常系/異常系のレスポンス処理
- ローディング状態の表示テスト
- エラーハンドリングテスト
- キャッシュ動作の検証

## パフォーマンステスト

### Core Web Vitals
- LCP (Largest Contentful Paint) < 2.5s
- FID (First Input Delay) < 100ms
- CLS (Cumulative Layout Shift) < 0.1

### Bundle Size Monitoring
- webpack-bundle-analyzer による依存関係分析
- Code splitting の効果測定
- Tree shaking の最適化確認
