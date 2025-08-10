# Kotlin 開発能力

Kotlin を使用した現代的なバックエンド・アプリケーション開発能力。

以下は、堅牢で良いとされる書き方や、より良い（better）とされる書き方の例である。これらのパターンを参考にし、コード生成時に採用すること。

---

## Null 安全の活用

**Bad**

```kotlin
val name: String = user.name // NullPointerException のリスク
```

**Better**

```kotlin
val name: String = user.name ?: "Unknown" // デフォルト値を設定
```

> `?:` 演算子で null 安全性を担保する。

---

## 不変データクラスの使用

**Bad**

```kotlin
class User(var name: String, var age: Int)
```

**Better**

```kotlin
data class User(val name: String, val age: Int)
```

> 不変プロパティを使うことで、スレッドセーフで予測可能な動作になる。

---

## when 式の exhaustive 使用

**Bad**

```kotlin
when (status) {
    Status.ACTIVE -> handleActive()
    Status.INACTIVE -> handleInactive()
}
```

**Better**

```kotlin
when (status) {
    Status.ACTIVE -> handleActive()
    Status.INACTIVE -> handleInactive()
    Status.PENDING -> handlePending()
}
```

> sealed class や enum は `when` で網羅性を保証する。

---

## 拡張関数の活用

**Bad**

```kotlin
fun calculateDiscount(price: Int, percentage: Int): Int {
    return price - (price * percentage / 100)
}
```

**Better**

```kotlin
fun Int.discount(percentage: Int): Int = this - (this * percentage / 100)

val finalPrice = 2000.discount(10)
```

> 拡張関数で可読性を向上。

---

## スコープ関数の適切な使用

**Bad**

```kotlin
val user = User("Alice", 25)
user.age += 1
println(user)
```

**Better**

```kotlin
val user = User("Alice", 25).apply { age += 1 }.also { println(it) }
```

> `apply` や `also` を適材適所で使い、初期化やデバッグを簡潔に。

---

## 非同期処理での構造化並行処理

**Bad**

```kotlin
GlobalScope.launch {
    delay(1000)
    println("Done")
}
```

**Better**

```kotlin
suspend fun fetchData() {
    coroutineScope {
        launch {
            delay(1000)
            println("Done")
        }
    }
}
```

> `GlobalScope` は避け、`coroutineScope` でキャンセル安全性を確保する。

---

## Result 型でのエラー管理

**Bad**

```kotlin
fun parseIntOrNull(input: String): Int? {
    return input.toIntOrNull()
}
```

**Better**

```kotlin
fun parseIntResult(input: String): Result<Int> {
    return runCatching { input.toInt() }
}
```

> 失敗時の情報を保持できる `Result` を利用する。

---
