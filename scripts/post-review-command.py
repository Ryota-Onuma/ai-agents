#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
post-review-command-improved.py — GitHub PRに「通常レビューコメント」と「インラインレビューコメント」を付けるPythonスクリプト

改良点:
- 一時ファイルを使用した長文コンテンツ対応
- PR作成者の自動判定(自分のPRの場合は制限回避)
- エラーハンドリングの強化
- レート制限対応

要件:
  - gh CLI がインストール & 認証済み(`gh auth login`)
  - 権限: Pull requests: write

使い方:
  # 1) 通常レビューコメント(レビューサマリー)
  python post-review-command-improved.py review  <pr> --body "text" [--approve | --request-changes | --comment]

  # 2) 通常コメント(Conversationタブの単発)
  python post-review-command-improved.py comment <pr> --body "text"

  # 3) インラインコメント(単一/複数行)
  python post-review-command-improved.py inline  <pr> --file <path> --line <n> [--start-line <m>] [--side RIGHT|LEFT] --body "text"

  # 4) インライン一括(レビューとしてまとめて送信)
  #    comments.json は {"event":"COMMENT","body":"optional summary","comments":[{...}]} 形式
  python post-review-command-improved.py inline-batch <pr> --json comments.json

PR指定(<pr>)は、PR番号 / PR URL / ブランチ名のいずれも可。
"""
from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from typing import Optional, Tuple, List, Dict, Any


API_VERSION = "2022-11-28"
MAX_RETRIES = 3
RETRY_DELAY = 2  # seconds


def run(cmd: List[str], *, capture_output: bool = True, text: bool = True, check: bool = True, input_data: Optional[str] = None) -> subprocess.CompletedProcess:
    """安全な subprocess.run ラッパー"""
    return subprocess.run(
        cmd,
        input=input_data,
        capture_output=capture_output,
        text=text,
        check=check,
    )


def require_gh():
    if shutil.which("gh") is None:
        sys.exit("Error: gh CLI が見つかりません。https://cli.github.com/ からインストールし、`gh auth login` を実行してください。")


@dataclass
class PRRef:
    owner: str
    repo: str
    number: int
    url: str


PR_URL_RE = re.compile(r"github\.com/([^/]+)/([^/]+)/pull/(\d+)")


def parse_pr_url(url: str) -> Optional[Tuple[str, str, int]]:
    m = PR_URL_RE.search(url)
    if not m:
        return None
    owner, repo, num = m.group(1), m.group(2), int(m.group(3))
    return owner, repo, num


def resolve_pr(pr: str) -> PRRef:
    """
    <pr> が番号/URL/ブランチ名のいずれでも、owner/repo/number/url に正規化する。
    まずURLとしてパースを試み、無理なら gh pr view でURLを取得してから再パース。
    """
    # 直接URLとして解決できるか
    if (t := parse_pr_url(pr)) is not None:
        owner, repo, number = t
        return PRRef(owner=owner, repo=repo, number=number, url=f"https://github.com/{owner}/{repo}/pull/{number}")

    # gh pr view で URL を取る
    # `--json url` はサポートされる(gh help formatting参照)
    cp = run(["gh", "pr", "view", pr, "--json", "url", "--jq", ".url"])
    url = cp.stdout.strip()
    if not url:
        sys.exit(f"Error: PR を解決できませんでした: {pr}")
    t = parse_pr_url(url)
    if t is None:
        sys.exit(f"Error: 取得したURLの解析に失敗しました: {url}")
    owner, repo, number = t
    return PRRef(owner=owner, repo=repo, number=number, url=url)


def get_head_sha(pr_spec: str) -> str:
    """
    PRのHEADコミットSHAを取得。`gh pr view --json headRefOid` が使える。
    """
    cp = run(["gh", "pr", "view", pr_spec, "--json", "headRefOid", "--jq", ".headRefOid"])
    sha = cp.stdout.strip()
    if not sha or not re.fullmatch(r"[0-9a-fA-F]{7,40}", sha):
        sys.exit("Error: PRのHEAD SHAを取得できませんでした(`gh pr view --json headRefOid`)。")
    return sha


def check_pr_author(pr_spec: str) -> bool:
    """
    PR作成者が自分かどうかをチェック
    """
    try:
        # PRの作成者情報を取得
        cp = run(["gh", "pr", "view", pr_spec, "--json", "author", "--jq", ".author.login"])
        if cp.returncode != 0:
            return False
        
        pr_author = cp.stdout.strip().strip('"')
        if not pr_author:
            return False
        
        # 現在のユーザー情報を取得
        user_cp = run(["gh", "api", "user", "--jq", ".login"])
        if user_cp.returncode != 0:
            return False
        
        current_user = user_cp.stdout.strip().strip('"')
        if not current_user:
            return False
        
        return pr_author == current_user
    except Exception:
        return False


def post_with_tempfile(content: str, pr_spec: str, review_type: str = "comment", state_flag: str = "--comment") -> bool:
    """
    一時ファイルを使って長文コンテンツを投稿
    """
    with tempfile.NamedTemporaryFile(mode='w', suffix='.md', delete=False, encoding='utf-8') as f:
        f.write(content)
        temp_path = f.name

    try:
        if review_type == "review":
            cmd = ["gh", "pr", "review", pr_spec, state_flag, "--body-file", temp_path]
        else:
            cmd = ["gh", "pr", "comment", pr_spec, "--body-file", temp_path]

        result = run(cmd, capture_output=False, check=False)
        if result.returncode == 0:
            print(f"✅ {review_type} の投稿が完了しました")
            return True
        else:
            print(f"❌ {review_type} の投稿に失敗しました: {result.stderr}")
            return False
    finally:
        os.unlink(temp_path)


def retry_with_backoff(func, *args, **kwargs):
    """
    指数バックオフ付きリトライ
    """
    for attempt in range(MAX_RETRIES):
        try:
            return func(*args, **kwargs)
        except subprocess.CalledProcessError as e:
            if "rate limit" in (e.stderr or "").lower() and attempt < MAX_RETRIES - 1:
                wait_time = RETRY_DELAY * (2 ** attempt)
                print(f"⚠️ レート制限により待機中... {wait_time}秒後にリトライします")
                time.sleep(wait_time)
                continue
            else:
                raise
        except Exception as e:
            if attempt < MAX_RETRIES - 1:
                wait_time = RETRY_DELAY * (2 ** attempt)
                print(f"⚠️ エラーが発生しました。{wait_time}秒後にリトライします: {e}")
                time.sleep(wait_time)
                continue
            else:
                raise


# ----- サブコマンド実装 -----

def cmd_review(args: argparse.Namespace) -> None:
    if not args.body:
        sys.exit("Error: --body は必須です。")

    # PR作成者チェック
    is_own_pr = check_pr_author(args.pr)
    
    state_flag = "--comment"
    if args.approve:
        state_flag = "--approve"
    elif args.request_changes:
        if is_own_pr:
            print("⚠️ 自分のPRには --request-changes は使用できません。通常のコメントとして投稿します。")
            state_flag = "--comment"
        else:
            state_flag = "--request-changes"

    # 一時ファイルを使用して投稿
    success = post_with_tempfile(args.body, args.pr, "review", state_flag)
    if not success:
        sys.exit(1)


def cmd_comment(args: argparse.Namespace) -> None:
    if not args.body:
        sys.exit("Error: --body は必須です。")
    
    success = post_with_tempfile(args.body, args.pr, "comment")
    if not success:
        sys.exit(1)


def cmd_inline(args: argparse.Namespace) -> None:
    if not args.file or not args.line or not args.body:
        sys.exit("Error: --file / --line / --body は必須です。")
    if args.side not in ("RIGHT", "LEFT"):
        sys.exit("Error: --side は RIGHT / LEFT のみ有効です。")

    prref = resolve_pr(args.pr)
    head_sha = get_head_sha(args.pr)

    # gh api POST /repos/{owner}/{repo}/pulls/{pull_number}/comments
    # line/side (+ start_line/start_side) で複数行に対応
    base_cmd = [
        "gh", "api",
        f"repos/{prref.owner}/{prref.repo}/pulls/{prref.number}/comments",
        "-X", "POST",
        "-H", "Accept: application/vnd.github+json",
        "-H", f"X-GitHub-Api-Version: {API_VERSION}",
        "-F", f"body={args.body}",
        "-F", f"commit_id={head_sha}",
        "-F", f"path={args.file}",
        "-F", f"line={args.line}",
        "-F", f"side={args.side}",
    ]
    if args.start_line is not None:
        base_cmd.extend(["-F", f"start_line={args.start_line}", "-F", f"start_side={args.side}"])

    try:
        retry_with_backoff(run, base_cmd, capture_output=False, check=True)
        print("✅ インラインコメントの投稿が完了しました")
    except subprocess.CalledProcessError as e:
        print(f"❌ インラインコメントの投稿に失敗しました: {e}")
        sys.exit(1)


def cmd_inline_batch(args: argparse.Namespace) -> None:
    if not args.json:
        sys.exit("Error: --json <file> が必要です。")
    if not os.path.isfile(args.json):
        sys.exit(f"Error: JSONファイルが見つかりません: {args.json}")

    # JSONのバリデーション(最低限)
    try:
        with open(args.json, "r", encoding="utf-8") as f:
            payload = json.load(f)
        if "event" not in payload or "comments" not in payload:
            sys.exit("Error: JSONは {\"event\":\"COMMENT|APPROVE|REQUEST_CHANGES\", \"comments\":[...]} を含む必要があります。")
    except json.JSONDecodeError as e:
        sys.exit(f"Error: JSONの読み込みに失敗しました: {e}")

    prref = resolve_pr(args.pr)
    
    # 自分のPRの場合はeventを調整
    is_own_pr = check_pr_author(args.pr)
    if is_own_pr and payload.get("event") == "REQUEST_CHANGES":
        payload["event"] = "COMMENT"
        print("⚠️ 自分のPRのため、REQUEST_CHANGESをCOMMENTに変更しました")

    # 一時ファイルにJSONを保存
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False, encoding='utf-8') as f:
        json.dump(payload, f, indent=2, ensure_ascii=False)
        json_path = f.name

    try:
        # NOTE: コメント各要素に commit_id を含めない場合、GitHub側でPRの最新HEADが用いられる仕様
        # 必要に応じ comments[i].commit_id を明示指定してください。
        api_cmd = [
            "gh", "api",
            f"repos/{prref.owner}/{prref.repo}/pulls/{prref.number}/reviews",
            "-X", "POST",
            "-H", "Accept: application/vnd.github+json",
            "-H", f"X-GitHub-Api-Version: {API_VERSION}",
            "--input", json_path  # 一時ファイルからpayloadを渡す
        ]
        
        try:
            retry_with_backoff(run, api_cmd, capture_output=False, check=True)
            print("✅ インラインコメント一括投稿が完了しました")
        except subprocess.CalledProcessError as e:
            print(f"❌ インラインコメント一括投稿に失敗しました: {e}")
            sys.exit(1)
    finally:
        os.unlink(json_path)


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="GitHub PRレビュー支援(通常レビュー/コメント + インラインコメント) - 改良版")
    sub = p.add_subparsers(dest="subcmd", required=True)

    # review
    sp = sub.add_parser("review", help="通常レビューコメント(approve/request-changes/comment)を送信")
    sp.add_argument("pr", help="PR番号/URL/ブランチ名")
    sp.add_argument("--body", required=True, help="本文")
    g = sp.add_mutually_exclusive_group()
    g.add_argument("--approve", action="store_true", help="Approveとして送信")
    g.add_argument("--request-changes", action="store_true", help="Request changesとして送信")
    g.add_argument("--comment", action="store_true", help="Comment(デフォルト)として送信")
    sp.set_defaults(func=cmd_review)

    # comment
    sp = sub.add_parser("comment", help="通常コメント(Conversationタブの単発コメント)を送信")
    sp.add_argument("pr", help="PR番号/URL/ブランチ名")
    sp.add_argument("--body", required=True, help="本文")
    sp.set_defaults(func=cmd_comment)

    # inline
    sp = sub.add_parser("inline", help="インラインレビューコメントを1件送信")
    sp.add_argument("pr", help="PR番号/URL/ブランチ名")
    sp.add_argument("--file", required=True, help="ファイルパス(PRのdiff上のパス)")
    sp.add_argument("--line", required=True, type=int, help="対象の最終行番号(diff上の行)")
    sp.add_argument("--start-line", type=int, help="複数行コメントの先頭行(省略時は単一行)")
    sp.add_argument("--side", default="RIGHT", choices=["RIGHT", "LEFT"], help="コメント側(通常はRIGHT)")
    sp.add_argument("--body", required=True, help="本文")
    sp.set_defaults(func=cmd_inline)

    # inline-batch
    sp = sub.add_parser("inline-batch", help="インラインコメントを複数まとめてレビューとして送信")
    sp.add_argument("pr", help="PR番号/URL/ブランチ名")
    sp.add_argument("--json", required=True, help="comments付きのJSONファイルパス")
    sp.set_defaults(func=cmd_inline_batch)

    return p


def main():
    require_gh()
    parser = build_parser()
    args = parser.parse_args()
    
    try:
        args.func(args)
    except subprocess.CalledProcessError as e:
        # gh / API 側のエラー表示をそのまま標準エラーに流す
        sys.stderr.write(e.stderr or "")
        sys.exit(e.returncode)
    except KeyboardInterrupt:
        print("\n⚠️ ユーザーによって中断されました")
        sys.exit(1)
    except Exception as e:
        print(f"❌ 予期しないエラーが発生しました: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
