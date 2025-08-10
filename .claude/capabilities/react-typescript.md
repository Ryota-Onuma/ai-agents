# React + TypeScript 開発能力

React と TypeScript を組み合わせた現代的なフロントエンド開発能力。

以下は、堅牢で良いとされる書き方や、より良い（better）とされる書き方の few-shot 例である。Claude はこれらのパターンを参考にし、コード生成時に採用すること。

---

## Props 型定義は明示し、`React.FC` は原則使わない

**Bad**

```tsx
const Button = (props) => <button>{props.label}</button>;
```

**Better**

```tsx
type ButtonProps = {
  label: string;
  onClick?: () => void;
  disabled?: boolean;
};

const Button = ({ label, onClick, disabled }: ButtonProps) => {
  return (
    <button onClick={onClick} disabled={disabled}>
      {label}
    </button>
  );
};
```

> `React.FC` は `children` を暗黙に許可しがちで誤用を招く。必要時のみ明示的に `children` を型定義する。

---

## Discriminated Union で状態を表現（exhaustive 保証）

**Bad**

```tsx
type FetchState<T> = {
  status: string; // "idle" | "loading" | "success" | "failure" の想定
  data?: T;
  message?: string;
};
```

**Better**

```tsx
type Idle = { kind: "idle" };
type Loading = { kind: "loading" };
type Success<T> = { kind: "success"; data: T };
type Failure = { kind: "failure"; message: string };

type FetchState<T> = Idle | Loading | Success<T> | Failure;

const View = <T,>({ state }: { state: FetchState<T> }) => {
  switch (state.kind) {
    case "idle":
      return <p>Idle</p>;
    case "loading":
      return <p>Loading...</p>;
    case "success":
      return <pre>{JSON.stringify(state.data, null, 2)}</pre>;
    case "failure":
      return <p role="alert">{state.message}</p>;
    default: {
      const _exhaustive: never = state; // 将来の分岐漏れを検出
      return _exhaustive;
    }
  }
};
```

> `kind`（判別可能プロパティ）で分岐し `never` チェックで網羅性を担保。

---

## `children` は必要なときだけ受け取る

**Bad**

```tsx
type CardProps = { title: string };
const Card: React.FC<CardProps> = ({ title, children }) => (
  <section>
    <h2>{title}</h2>
    <div>{children}</div>
  </section>
);
```

**Better**

```tsx
type CardProps = {
  title: string;
  children: React.ReactNode; // children が必要なときだけ定義
};

const Card = ({ title, children }: CardProps) => {
  return (
    <section>
      <h2>{title}</h2>
      <div>{children}</div>
    </section>
  );
};
```

> `children` の受け取りは意図的に。不要なコンポーネントには持たせない。

---

## イベントハンドラの型は推論 or 公式型を使用

**Bad**

```tsx
const Search = ({ onChange }: { onChange: (e: any) => void }) => {
  return <input onChange={onChange} />;
};
```

**Better**

```tsx
const Search = ({
  onChange,
}: {
  onChange: React.ChangeEventHandler<HTMLInputElement>;
}) => {
  return <input onChange={onChange} />;
};
// あるいは、呼び出し側で推論させる
<input onChange={(e) => setQuery(e.currentTarget.value)} />;
```

> `any` は避け、`ChangeEventHandler<HTMLInputElement>` 等の公式型を使うか推論に任せる。

---

## Null/Undefined の明示（`?` と `??` / `?.`）

**Bad**

```tsx
const email: string = user.profile.email; // 例外の元
```

**Better**

```tsx
const email = user.profile?.email ?? ""; // 空文字デフォルト
```

> optional chaining / null 合体で安全に値を扱う。

---

## `useEffect` 依存は正しく管理（関数は `useCallback`）

**Bad**

```tsx
useEffect(() => {
  fetchUsers(); // 毎回新しい参照で無限実行の恐れ
});
```

**Better**

```tsx
const fetchUsers = useCallback(async () => {
  /* ... */
}, []);

useEffect(() => {
  fetchUsers();
}, [fetchUsers]);
```

> 依存関係は ESLint の `react-hooks/exhaustive-deps` に従う。関数は必要なときだけメモ化。

---

## useCallback のユースケースと指針

**要点**

- **使うと良い場面は主に 3 つ**：

  1. `React.memo` された子へ関数を **props** として渡すとき
  2. `useEffect` / `useMemo` の **依存**として関数を渡す必要があるとき
  3. **カスタムフック**で安定した関数を返すとき

- それ以外では **乱用しない**。初回レンダーで関数を包むコストや可読性低下のデメリットがある。
- 依存配列には **関数内で参照する reactive な値（state/props/外部変数）をすべて**入れる。

**ユースケース 1：memo 化された子への関数渡し**

```tsx
const Parent = () => {
  const handleClick = useCallback(() => {
    console.log("clicked");
  }, []); // 参照を安定化
  return <Child onClick={handleClick} />;
};

const Child = React.memo(({ onClick }: { onClick: () => void }) => {
  console.log("Child render");
  return <button onClick={onClick}>Click me</button>;
});
// → 子は props が同一参照なら再レンダリングされない
```

**ユースケース 2：useEffect の依存として使う**

```tsx
const Component = ({ value }: { value: number }) => {
  const doSomething = useCallback(() => {
    console.log("value:", value);
  }, [value]); // 参照する value を依存に含める

  useEffect(() => {
    doSomething();
  }, [doSomething]);

  return null;
};
```

**ユースケース 3：カスタムフックで安定した関数を返す**

```tsx
const useCounter = () => {
  const [count, setCount] = useState(0);
  const increment = useCallback(() => {
    setCount((c) => c + 1);
  }, []);
  return { count, increment };
};

const Counter = () => {
  const { count, increment } = useCounter();
  return <button onClick={increment}>{count}</button>;
};
```

**アンチパターン（避ける）**

- 毎回再生成でも**問題ない関数**にまで機械的に `useCallback` を付ける
- 依存配列に必要な値を入れず、**stale な参照**になる

---

## `useMemo` は高コスト計算に限定（過剰使用しない）

**Bad**

```tsx
const doubled = useMemo(() => value * 2, [value]); // ただの算術
```

**Better**

```tsx
const expensive = useMemo(() => heavyCompute(items), [items]);
```

> 小さな式のメモ化は逆効果。重い計算や安定した参照が必要なときのみ。

---

## リストの `key` は安定したユニーク ID を使用

**Bad**

```tsx
{
  items.map((it, i) => <li key={i}>{it.label}</li>);
}
```

**Better**

```tsx
{
  items.map((it) => <li key={it.id}>{it.label}</li>);
}
```

> インデックス `key` は並び替えで破綻。ID を使う。

---

## フォームは型安全に（TypeScript の `asserts`）

**Bad**

```tsx
// バリデーションと型が二重管理 / any の氾濫
```

**Better**

```tsx
type FormValues = { email: string; age: number };

const assertFormValues = (x: unknown): asserts x is FormValues => {
  if (typeof x !== "object" || x === null) throw new Error("invalid form");
  const o = x as any;
  if (typeof o.email !== "string" || !o.email.includes("@"))
    throw new Error("invalid email");
  if (typeof o.age !== "number" || !Number.isInteger(o.age) || o.age < 18)
    throw new Error("invalid age");
};

const SignupForm = () => {
  const [error, setError] = React.useState<string | null>(null);

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const values = {
      email: String(fd.get("email") ?? ""),
      age: Number(fd.get("age")),
    };
    try {
      assertFormValues(values); // ここで型が FormValues に絞り込まれる
      // values は厳密な型で以後利用可能
      console.log(values);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "validation error");
    }
  };

  return (
    <form onSubmit={onSubmit}>
      <input name="email" type="email" />
      <input name="age" type="number" />
      {error && <span role="alert">{error}</span>}
      <button type="submit">Sign up</button>
    </form>
  );
};
```

> `asserts x is T` で実行時検証と型の絞り込みを一致させ、**単一の真実の源泉**を保つ。

---

## サーバー状態とローカル状態を分離（React Query の例）

**Bad**

```tsx
// fetch と useState を組み合わせて再実装
```

**Better**

```tsx
import { useQuery } from "@tanstack/react-query";

const Users = () => {
  const { data, isLoading, error } = useQuery({
    queryKey: ["users"],
    queryFn: async () => (await fetch("/api/users")).json(),
    staleTime: 60_000,
  });

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p role="alert">Error</p>;
  return (
    <ul>
      {data.map((u: { id: string; name: string }) => (
        <li key={u.id}>{u.name}</li>
      ))}
    </ul>
  );
};
```

> キャッシュ・リトライ・並列などの責務をライブラリに委譲。

---

## CSS とスタイルの原則（例）

**Bad**

```tsx
<div style={{ padding: 8, color: "#333" }}>Text</div>
```

**Better**

```tsx
// Tailwind or CSS Modules など、プロジェクト標準に従う
<div className="p-2 text-neutral-700">Text</div>
```

> インライン乱用は避け、再利用とテーマ適用を容易にする。

---

## ディレクトリと命名（一例）

```
src/
  components/    // 再利用コンポーネント
  features/      // ドメイン単位（画面/機能ごと）
  hooks/         // 再利用フック
  lib/           // util, API クライアント
  pages/         // ルーティング（Next.js 等）
  types/         // 共有型
```

> ドメインごとに関心の分離を行い、循環依存を避ける。

---

## ESLint/TSConfig（要点）

- `strict: true` は必須
- `noUncheckedIndexedAccess: true` で配列/レコードアクセスを安全化
- `exactOptionalPropertyTypes: true` で `?` の意味を厳密化
- `eslint-config-next` / `@typescript-eslint` / `eslint-plugin-react-hooks` を採用

---

### まとめ

- **型を先に**：Union と `never` で網羅性を保証
- **状態の責務分離**：サーバー状態はライブラリ、ローカルは React
- **フック規律**：依存配列とメモ化は最小限・正確に
- **実行時検証**：TypeScript の type guard / `asserts` で API 境界を堅牢化

# React + TypeScript 開発能力

React と TypeScript を組み合わせた現代的なフロントエンド開発能力。

以下は、堅牢で良いとされる書き方や、より良い（better）とされる書き方の few-shot 例である。Claude はこれらのパターンを参考にし、コード生成時に採用すること。

---

## Props 型定義は明示し、`React.FC` は原則使わない

**Bad**

```tsx
const Button = (props) => <button>{props.label}</button>;
```

**Better**

```tsx
type ButtonProps = {
  label: string;
  onClick?: () => void;
  disabled?: boolean;
};

const Button = ({ label, onClick, disabled }: ButtonProps) => {
  return (
    <button onClick={onClick} disabled={disabled}>
      {label}
    </button>
  );
};
```

> `React.FC` は `children` を暗黙に許可しがちで誤用を招く。必要時のみ明示的に `children` を型定義する。

---

## Discriminated Union で状態を表現（exhaustive 保証）

**Bad**

```tsx
type FetchState<T> = {
  status: string; // "idle" | "loading" | "success" | "failure" の想定
  data?: T;
  message?: string;
};
```

**Better**

```tsx
type Idle = { kind: "idle" };
type Loading = { kind: "loading" };
type Success<T> = { kind: "success"; data: T };
type Failure = { kind: "failure"; message: string };

type FetchState<T> = Idle | Loading | Success<T> | Failure;

const View = <T,>({ state }: { state: FetchState<T> }) => {
  switch (state.kind) {
    case "idle":
      return <p>Idle</p>;
    case "loading":
      return <p>Loading...</p>;
    case "success":
      return <pre>{JSON.stringify(state.data, null, 2)}</pre>;
    case "failure":
      return <p role="alert">{state.message}</p>;
    default: {
      const _exhaustive: never = state; // 将来の分岐漏れを検出
      return _exhaustive;
    }
  }
};
```

> `kind`（判別可能プロパティ）で分岐し `never` チェックで網羅性を担保。

---

## `children` は必要なときだけ受け取る

**Bad**

```tsx
type CardProps = { title: string };
const Card: React.FC<CardProps> = ({ title, children }) => (
  <section>
    <h2>{title}</h2>
    <div>{children}</div>
  </section>
);
```

**Better**

```tsx
type CardProps = {
  title: string;
  children: React.ReactNode; // children が必要なときだけ定義
};

const Card = ({ title, children }: CardProps) => {
  return (
    <section>
      <h2>{title}</h2>
      <div>{children}</div>
    </section>
  );
};
```

> `children` の受け取りは意図的に。不要なコンポーネントには持たせない。

---

## イベントハンドラの型は推論 or 公式型を使用

**Bad**

```tsx
const Search = ({ onChange }: { onChange: (e: any) => void }) => {
  return <input onChange={onChange} />;
};
```

**Better**

```tsx
const Search = ({
  onChange,
}: {
  onChange: React.ChangeEventHandler<HTMLInputElement>;
}) => {
  return <input onChange={onChange} />;
};
// あるいは、呼び出し側で推論させる
<input onChange={(e) => setQuery(e.currentTarget.value)} />;
```

> `any` は避け、`ChangeEventHandler<HTMLInputElement>` 等の公式型を使うか推論に任せる。

---

## Null/Undefined の明示（`?` と `??` / `?.`）

**Bad**

```tsx
const email: string = user.profile.email; // 例外の元
```

**Better**

```tsx
const email = user.profile?.email ?? ""; // 空文字デフォルト
```

> optional chaining / null 合体で安全に値を扱う。

---

## `useEffect` 依存は正しく管理（関数は `useCallback`）

**Bad**

```tsx
useEffect(() => {
  fetchUsers(); // 毎回新しい参照で無限実行の恐れ
});
```

**Better**

```tsx
const fetchUsers = useCallback(async () => {
  /* ... */
}, []);

useEffect(() => {
  fetchUsers();
}, [fetchUsers]);
```

> 依存関係は ESLint の `react-hooks/exhaustive-deps` に従う。関数は必要なときだけメモ化。

---

## `useMemo` は高コスト計算に限定（過剰使用しない）

**Bad**

```tsx
const doubled = useMemo(() => value * 2, [value]); // ただの算術
```

**Better**

```tsx
const expensive = useMemo(() => heavyCompute(items), [items]);
```

> 小さな式のメモ化は逆効果。重い計算や安定した参照が必要なときのみ。

---

## リストの `key` は安定したユニーク ID を使用

**Bad**

```tsx
{
  items.map((it, i) => <li key={i}>{it.label}</li>);
}
```

**Better**

```tsx
{
  items.map((it) => <li key={it.id}>{it.label}</li>);
}
```

> インデックス `key` は並び替えで破綻。ID を使う。

---

## フォームは型安全に（TypeScript の `asserts`）

**Bad**

```tsx
// バリデーションと型が二重管理 / any の氾濫
```

**Better**

```tsx
type FormValues = { email: string; age: number };

const assertFormValues = (x: unknown): asserts x is FormValues => {
  if (typeof x !== "object" || x === null) throw new Error("invalid form");
  const o = x as any;
  if (typeof o.email !== "string" || !o.email.includes("@"))
    throw new Error("invalid email");
  if (typeof o.age !== "number" || !Number.isInteger(o.age) || o.age < 18)
    throw new Error("invalid age");
};

const SignupForm = () => {
  const [error, setError] = React.useState<string | null>(null);

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const values = {
      email: String(fd.get("email") ?? ""),
      age: Number(fd.get("age")),
    };
    try {
      assertFormValues(values); // ここで型が FormValues に絞り込まれる
      // values は厳密な型で以後利用可能
      console.log(values);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "validation error");
    }
  };

  return (
    <form onSubmit={onSubmit}>
      <input name="email" type="email" />
      <input name="age" type="number" />
      {error && <span role="alert">{error}</span>}
      <button type="submit">Sign up</button>
    </form>
  );
};
```

> `asserts x is T` で実行時検証と型の絞り込みを一致させ、**単一の真実の源泉**を保つ。

---

## サーバー状態とローカル状態を分離（React Query の例）

**Bad**

```tsx
// fetch と useState を組み合わせて再実装
```

**Better**

```tsx
import { useQuery } from "@tanstack/react-query";

const Users = () => {
  const { data, isLoading, error } = useQuery({
    queryKey: ["users"],
    queryFn: async () => (await fetch("/api/users")).json(),
    staleTime: 60_000,
  });

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p role="alert">Error</p>;
  return (
    <ul>
      {data.map((u: { id: string; name: string }) => (
        <li key={u.id}>{u.name}</li>
      ))}
    </ul>
  );
};
```

> キャッシュ・リトライ・並列などの責務をライブラリに委譲。

---

## CSS とスタイルの原則（例）

**Bad**

```tsx
<div style={{ padding: 8, color: "#333" }}>Text</div>
```

**Better**

```tsx
// Tailwind or CSS Modules など、プロジェクト標準に従う
<div className="p-2 text-neutral-700">Text</div>
```

> インライン乱用は避け、再利用とテーマ適用を容易にする。

---

## ディレクトリと命名（一例）

```
src/
  components/    // 再利用コンポーネント
  features/      // ドメイン単位（画面/機能ごと）
  hooks/         // 再利用フック
  lib/           // util, API クライアント
  pages/         // ルーティング（Next.js 等）
  types/         // 共有型
```

> ドメインごとに関心の分離を行い、循環依存を避ける。

---

## ESLint/TSConfig（要点）

- `strict: true` は必須
- `noUncheckedIndexedAccess: true` で配列/レコードアクセスを安全化
- `exactOptionalPropertyTypes: true` で `?` の意味を厳密化
- `eslint-config-next` / `@typescript-eslint` / `eslint-plugin-react-hooks` を採用

---

### まとめ

- **型を先に**：Union と `never` で網羅性を保証
- **状態の責務分離**：サーバー状態はライブラリ、ローカルは React
- **フック規律**：依存配列とメモ化は最小限・正確に
- **実行時検証**：TypeScript の type guard / `asserts` で API 境界を堅牢化
