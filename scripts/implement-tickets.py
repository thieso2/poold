#!/usr/bin/env python3
"""Implement GitHub issues in dependency-ordered, parallel worktree batches."""

from __future__ import annotations

import argparse
import concurrent.futures
import dataclasses
import datetime
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
from collections.abc import Sequence


@dataclasses.dataclass(frozen=True)
class Ticket:
    number: int
    title: str
    blockers: tuple[int, ...]


@dataclasses.dataclass(frozen=True)
class TicketJob:
    ticket: Ticket
    worktree: pathlib.Path
    branch: str
    log: pathlib.Path


class OrchestrationError(RuntimeError):
    pass


def run(
    args: Sequence[str],
    *,
    cwd: pathlib.Path,
    capture: bool = True,
    check: bool = True,
    output: pathlib.Path | None = None,
) -> subprocess.CompletedProcess[str]:
    if output is not None:
        with output.open("w", encoding="utf-8") as stream:
            return subprocess.run(
                args,
                cwd=cwd,
                text=True,
                stdout=stream,
                stderr=subprocess.STDOUT,
                check=check,
            )
    return subprocess.run(
        args,
        cwd=cwd,
        text=True,
        capture_output=capture,
        check=check,
    )


def command_output(args: Sequence[str], cwd: pathlib.Path) -> str:
    return run(args, cwd=cwd).stdout.strip()


def worktree_status(root: pathlib.Path) -> str:
    return command_output(
        ["git", "status", "--porcelain", "--untracked-files=all"], root
    )


def run_codex_agent(
    worktree: pathlib.Path, prompt: str, log: pathlib.Path
) -> subprocess.CompletedProcess[str]:
    return run(
        [
            "codex",
            "exec",
            "--sandbox",
            "danger-full-access",
            "--cd",
            str(worktree),
            prompt,
        ],
        cwd=worktree,
        check=False,
        output=log,
    )


def repository_root() -> pathlib.Path:
    try:
        root = command_output(
            ["git", "rev-parse", "--show-toplevel"], pathlib.Path.cwd()
        )
    except subprocess.CalledProcessError as error:
        raise OrchestrationError("run this script inside a Git repository") from error
    return pathlib.Path(root).resolve()


def repository_name(root: pathlib.Path) -> str:
    result = command_output(
        ["gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner"],
        root,
    )
    if not result:
        raise OrchestrationError("could not infer the GitHub repository")
    return result


def selected_issue_numbers(
    root: pathlib.Path, repository: str, requested: Sequence[int]
) -> list[int]:
    if requested:
        return sorted(set(requested))
    raw = command_output(
        [
            "gh",
            "issue",
            "list",
            "--repo",
            repository,
            "--state",
            "open",
            "--label",
            "ready-for-agent",
            "--limit",
            "1000",
            "--json",
            "number",
            "--jq",
            ".[].number",
        ],
        root,
    )
    return sorted(int(line) for line in raw.splitlines() if line)


def issue_json(root: pathlib.Path, repository: str, number: int) -> dict:
    raw = command_output(
        [
            "gh",
            "issue",
            "view",
            str(number),
            "--repo",
            repository,
            "--comments",
            "--json",
            "number,title,state,labels,comments",
            "--jq",
            (
                "{number,title,state,labels:[.labels[].name],"
                "comments:[.comments[].body]}"
            ),
        ],
        root,
    )
    return json.loads(raw)


def blocker_json(root: pathlib.Path, repository: str, number: int) -> list[dict]:
    raw = command_output(
        [
            "gh",
            "api",
            f"repos/{repository}/issues/{number}/dependencies/blocked_by?per_page=100",
        ],
        root,
    )
    return json.loads(raw)


def load_tickets(
    root: pathlib.Path, repository: str, numbers: Sequence[int]
) -> dict[int, Ticket]:
    tickets: dict[int, Ticket] = {}
    selected = set(numbers)
    for number in numbers:
        issue = issue_json(root, repository, number)
        if issue["state"].upper() != "OPEN":
            raise OrchestrationError(f"issue #{number} is not open")
        blockers = blocker_json(root, repository, number)
        open_blockers = {
            int(blocker["number"])
            for blocker in blockers
            if blocker["state"].lower() == "open"
        }
        outside = sorted(open_blockers - selected)
        if outside:
            formatted = ", ".join(f"#{blocker}" for blocker in outside)
            raise OrchestrationError(
                f"issue #{number} has open blockers outside this run: {formatted}"
            )
        tickets[number] = Ticket(
            number=number,
            title=issue["title"],
            blockers=tuple(sorted(open_blockers)),
        )
    return tickets


def dependency_batches(tickets: dict[int, Ticket]) -> list[list[Ticket]]:
    remaining = dict(tickets)
    completed: set[int] = set()
    batches: list[list[Ticket]] = []
    while remaining:
        ready = sorted(
            (
                ticket
                for ticket in remaining.values()
                if set(ticket.blockers) <= completed
            ),
            key=lambda ticket: ticket.number,
        )
        if not ready:
            cycle = ", ".join(f"#{number}" for number in sorted(remaining))
            raise OrchestrationError(
                f"dependency cycle or unresolved dependency among: {cycle}"
            )
        batches.append(ready)
        for ticket in ready:
            completed.add(ticket.number)
            del remaining[ticket.number]
    return batches


def ensure_prerequisites(root: pathlib.Path, main_branch: str) -> str:
    for executable in ("codex", "gh", "git"):
        if shutil.which(executable) is None:
            raise OrchestrationError(f"required executable is missing: {executable}")
    current = command_output(["git", "branch", "--show-current"], root)
    if current != main_branch:
        raise OrchestrationError(
            f"checkout {main_branch!r} before running (currently on {current!r})"
        )
    tracked_changes = command_output(
        ["git", "status", "--porcelain", "--untracked-files=no"], root
    )
    if tracked_changes:
        raise OrchestrationError("the main worktree has tracked changes")
    merge_state = root / ".git" / "MERGE_HEAD"
    if merge_state.exists():
        raise OrchestrationError("the main worktree already has a merge in progress")
    return worktree_status(root)


def safe_name(value: str) -> str:
    return re.sub(r"[^A-Za-z0-9._-]+", "-", value).strip("-")


def ticket_prompt(ticket: Ticket) -> str:
    return (
        f"$implement #{ticket.number}\n\n"
        "Implement this ticket completely in the current worktree. Read the full "
        "GitHub issue, its comments, linked specification, AGENTS.md, CONTEXT.md, "
        "and relevant ADRs. Follow the implement skill faithfully, including TDD "
        "at the issue's public acceptance seams and its independent code review. "
        "Run focused checks during development and the full test suite once at the "
        "end. Commit only this ticket's intended changes on the current branch. "
        "Do not merge, push, close issues, or modify another worktree."
    )


def prepare_ticket_worktree(
    root: pathlib.Path,
    run_dir: pathlib.Path,
    batch_base: str,
    ticket: Ticket,
    run_id: str,
) -> TicketJob:
    branch = f"automation/issue-{ticket.number}-{run_id}"
    worktree = run_dir / f"issue-{ticket.number}"
    log = run_dir / f"issue-{ticket.number}.log"
    run(
        ["git", "worktree", "add", "-b", branch, str(worktree), batch_base],
        cwd=root,
        capture=False,
    )
    return TicketJob(ticket=ticket, worktree=worktree, branch=branch, log=log)


def run_ticket_agent(
    root: pathlib.Path,
    batch_base: str,
    job: TicketJob,
) -> TicketJob:
    completed = run_codex_agent(
        job.worktree, ticket_prompt(job.ticket), job.log
    )
    if completed.returncode != 0:
        raise OrchestrationError(
            f"agent for issue #{job.ticket.number} failed; inspect {job.log} "
            f"and {job.worktree}"
        )
    changes = worktree_status(job.worktree)
    if changes:
        raise OrchestrationError(
            f"agent for issue #{job.ticket.number} left uncommitted changes "
            f"in {job.worktree}"
        )
    commits = int(
        command_output(
            ["git", "rev-list", "--count", f"{batch_base}..{job.branch}"], root
        )
    )
    if commits < 1:
        raise OrchestrationError(
            f"agent for issue #{job.ticket.number} did not commit an implementation"
        )
    return job


def conflict_files(root: pathlib.Path) -> list[str]:
    raw = command_output(
        ["git", "diff", "--name-only", "--diff-filter=U"], root
    )
    return [line for line in raw.splitlines() if line]


def reconcile_merge_conflict(
    root: pathlib.Path, ticket: Ticket, log: pathlib.Path
) -> None:
    prompt = (
        f"Reconcile the in-progress merge of issue #{ticket.number} on main. "
        "Resolve every conflict by preserving the complete behavior from both "
        "main and the ticket branch. Read the issue and linked specification, run "
        "focused tests plus go test ./..., stage the resolutions, and complete the "
        "merge commit. Do not push or close the issue."
    )
    completed = run_codex_agent(root, prompt, log)
    if completed.returncode != 0 or conflict_files(root):
        raise OrchestrationError(
            f"merge reconciliation for issue #{ticket.number} failed; inspect {log}"
        )
    if (root / ".git" / "MERGE_HEAD").exists():
        raise OrchestrationError(
            f"merge reconciliation for issue #{ticket.number} did not commit"
        )


def merge_ticket(
    root: pathlib.Path,
    ticket: Ticket,
    branch: str,
    run_dir: pathlib.Path,
) -> None:
    merged = run(
        [
            "git",
            "merge",
            "--no-ff",
            branch,
            "-m",
            f"Merge issue #{ticket.number}: {ticket.title}",
        ],
        cwd=root,
        check=False,
    )
    if merged.returncode == 0:
        return
    if not conflict_files(root):
        raise OrchestrationError(
            f"merge of issue #{ticket.number} failed without conflicts:\n"
            f"{merged.stderr.strip()}"
        )
    reconcile_merge_conflict(
        root, ticket, run_dir / f"issue-{ticket.number}-merge.log"
    )


def reconcile_batch(
    root: pathlib.Path,
    tickets: Sequence[Ticket],
    run_dir: pathlib.Path,
    initial_status: str,
) -> None:
    issue_list = ", ".join(f"#{ticket.number}" for ticket in tickets)
    before = command_output(["git", "rev-parse", "HEAD"], root)
    prompt = (
        f"Reconcile the just-merged dependency batch {issue_list} on main. "
        "This is integration work, not a new ticket implementation. Inspect the "
        "combined diff and each issue's acceptance criteria for cross-ticket "
        "inconsistencies, duplicated approaches, or broken integration. Fix only "
        "integration problems, run focused checks and go test ./..., and commit "
        "any fixes with an imperative batch-reconciliation subject. If no fixes "
        "are needed, leave the worktree clean and do not create an empty commit. "
        "Do not push or close issues."
    )
    log = run_dir / f"batch-{'-'.join(str(ticket.number) for ticket in tickets)}.log"
    completed = run_codex_agent(root, prompt, log)
    if completed.returncode != 0:
        raise OrchestrationError(
            f"batch reconciliation failed; inspect {log}"
        )
    status = worktree_status(root)
    if status != initial_status:
        after = command_output(["git", "rev-parse", "HEAD"], root)
        detail = "without committing fixes" if before == after else "with new changes"
        raise OrchestrationError(
            f"batch reconciliation ended {detail}; inspect {log}"
        )
    run(["go", "test", "./..."], cwd=root, capture=False)


def remove_worktree(root: pathlib.Path, worktree: pathlib.Path) -> None:
    run(
        ["git", "worktree", "remove", "--force", str(worktree)],
        cwd=root,
        capture=False,
    )


def complete_issue_checklist(
    root: pathlib.Path, repository: str, number: int
) -> None:
    body = command_output(
        [
            "gh",
            "issue",
            "view",
            str(number),
            "--repo",
            repository,
            "--json",
            "body",
            "--jq",
            ".body",
        ],
        root,
    )
    completed = re.sub(
        r"^(\s*[-*+]\s+)\[ \]",
        r"\1[x]",
        body,
        flags=re.MULTILINE | re.IGNORECASE,
    )
    if completed == body:
        return
    with tempfile.NamedTemporaryFile(
        mode="w", encoding="utf-8", suffix=".md"
    ) as body_file:
        body_file.write(completed)
        body_file.flush()
        run(
            [
                "gh",
                "issue",
                "edit",
                str(number),
                "--repo",
                repository,
                "--body-file",
                body_file.name,
            ],
            cwd=root,
            capture=False,
        )


def finish_batch(
    root: pathlib.Path,
    repository: str,
    main_branch: str,
    tickets: Sequence[Ticket],
    push: bool,
    close: bool,
) -> None:
    if push:
        run(["git", "push", "origin", main_branch], cwd=root, capture=False)
    if close:
        if not push:
            raise OrchestrationError("--close requires pushing the merged batch")
        merged_sha = command_output(["git", "rev-parse", "--short=12", "HEAD"], root)
        for ticket in tickets:
            complete_issue_checklist(root, repository, ticket.number)
            run(
                [
                    "gh",
                    "issue",
                    "close",
                    str(ticket.number),
                    "--repo",
                    repository,
                    "--comment",
                    f"Implemented and reconciled on `{main_branch}` at {merged_sha}.",
                ],
                cwd=root,
                capture=False,
            )


def print_plan(batches: Sequence[Sequence[Ticket]]) -> None:
    for index, batch in enumerate(batches, start=1):
        rendered = " | ".join(
            f"#{ticket.number} {ticket.title}" for ticket in batch
        )
        print(f"batch {index}: {rendered}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Implement open GitHub tickets in dependency-ordered worktree batches. "
            "With no issue numbers, selects open ready-for-agent issues."
        )
    )
    parser.add_argument("issues", nargs="*", type=int, help="issue numbers")
    parser.add_argument(
        "--main-branch", default="main", help="integration branch (default: main)"
    )
    parser.add_argument(
        "--max-agents",
        type=int,
        default=4,
        help="maximum parallel ticket agents (default: 4)",
    )
    parser.add_argument(
        "--plan",
        action="store_true",
        help="print dependency batches without creating worktrees",
    )
    parser.add_argument(
        "--no-push",
        action="store_true",
        help="do not push main after each reconciled batch",
    )
    parser.add_argument(
        "--no-close",
        action="store_true",
        help="do not close successfully merged issues",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.max_agents < 1:
        raise OrchestrationError("--max-agents must be at least 1")
    if args.no_push and not args.no_close:
        raise OrchestrationError("--no-push requires --no-close")
    root = repository_root()
    repository = repository_name(root)
    numbers = selected_issue_numbers(root, repository, args.issues)
    if not numbers:
        print("No open tickets selected.")
        return 0
    tickets = load_tickets(root, repository, numbers)
    batches = dependency_batches(tickets)
    print_plan(batches)
    if args.plan:
        return 0

    initial_status = ensure_prerequisites(root, args.main_branch)
    run_id = safe_name(
        f"{datetime.datetime.now(datetime.UTC).strftime('%Y%m%dT%H%M%SZ')}-{os.getpid()}"
    )
    run_dir = pathlib.Path(
        tempfile.mkdtemp(prefix=f"poold-implement-{run_id}-")
    ).resolve()
    print(f"logs and active worktrees: {run_dir}")

    for index, batch in enumerate(batches, start=1):
        print(f"\nstarting batch {index}/{len(batches)}")
        batch_base = command_output(["git", "rev-parse", args.main_branch], root)
        results: list[TicketJob] = []
        with concurrent.futures.ThreadPoolExecutor(
            max_workers=min(args.max_agents, len(batch))
        ) as executor:
            prepared = [
                prepare_ticket_worktree(
                    root,
                    run_dir,
                    batch_base,
                    ticket,
                    run_id,
                )
                for ticket in batch
            ]
            futures = [
                executor.submit(
                    run_ticket_agent,
                    root,
                    batch_base,
                    ticket_worktree,
                )
                for ticket_worktree in prepared
            ]
            for future in concurrent.futures.as_completed(futures):
                results.append(future.result())

        results.sort(key=lambda result: result.ticket.number)
        for job in results:
            print(f"merging issue #{job.ticket.number} from {job.branch}")
            merge_ticket(root, job.ticket, job.branch, run_dir)
        reconcile_batch(root, batch, run_dir, initial_status)
        finish_batch(
            root,
            repository,
            args.main_branch,
            batch,
            push=not args.no_push,
            close=not args.no_close,
        )
        for job in results:
            remove_worktree(root, job.worktree)

    actions = ["implemented", "reconciled"]
    if not args.no_push:
        actions.append("pushed")
    if not args.no_close:
        actions.append("closed")
    print(f"\nAll selected tickets were {'; '.join(actions)}.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OrchestrationError, subprocess.CalledProcessError) as error:
        print(f"error: {error}", file=sys.stderr)
        raise SystemExit(1)
