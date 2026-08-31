#!/usr/bin/env python3
"""Verify provider JSON-null assumptions against peakhour-website-stable source."""

from __future__ import annotations

import ast
import json
import os
from pathlib import Path
import subprocess
import sys
from typing import Iterable


PROVIDER_ROOT = Path(__file__).resolve().parent.parent
CONTRACT_PATH = (
    PROVIDER_ROOT / "internal/provider/testdata/stable-json-null-contract.json"
)


def fail(message: str) -> None:
    raise AssertionError(message)


def parse_source(
    stable_root: Path, revision: str, relative_path: str
) -> ast.Module:
    result = subprocess.run(
        ["git", "-C", str(stable_root), "show", f"{revision}:{relative_path}"],
        check=True,
        capture_output=True,
        text=True,
    )
    return ast.parse(result.stdout, filename=f"{revision}:{relative_path}")


def find_definition(tree: ast.Module, kind: type[ast.AST], name: str) -> ast.AST:
    for node in ast.walk(tree):
        if isinstance(node, kind) and getattr(node, "name", None) == name:
            return node
    fail(f"could not find {kind.__name__} {name}")


def field_default(tree: ast.Module, class_name: str, field_name: str) -> ast.expr | None:
    class_node = find_definition(tree, ast.ClassDef, class_name)
    assert isinstance(class_node, ast.ClassDef)
    for node in class_node.body:
        if isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name):
            if node.target.id == field_name:
                return node.value
    fail(f"could not find {class_name}.{field_name}")


def require_constant(
    tree: ast.Module, class_name: str, field_name: str, expected: object
) -> None:
    default = field_default(tree, class_name, field_name)
    if not isinstance(default, ast.Constant) or default.value != expected:
        fail(
            f"{class_name}.{field_name} default is {ast.unparse(default) if default else '<required>'}, "
            f"want {expected!r}"
        )


def annotation_is_optional(annotation: ast.expr) -> bool:
    return any(
        isinstance(node, ast.Name) and node.id == "Optional"
        for node in ast.walk(annotation)
    )


def default_is_none(default: ast.expr | None) -> bool:
    if isinstance(default, ast.Constant):
        return default.value is None
    if isinstance(default, ast.Call) and isinstance(default.func, ast.Name):
        return (
            default.func.id == "Field"
            and bool(default.args)
            and isinstance(default.args[0], ast.Constant)
            and default.args[0].value is None
        )
    return False


def require_optional_fields_default_none(
    tree: ast.Module, class_names: set[str]
) -> None:
    for class_name in class_names:
        class_node = find_definition(tree, ast.ClassDef, class_name)
        assert isinstance(class_node, ast.ClassDef)
        for node in class_node.body:
            if not isinstance(node, ast.AnnAssign) or not isinstance(node.target, ast.Name):
                continue
            if annotation_is_optional(node.annotation) and not default_is_none(node.value):
                fail(
                    f"{class_name}.{node.target.id} is optional but does not default to None"
                )


def action_union_classes(tree: ast.Module) -> set[str]:
    for node in tree.body:
        if isinstance(node, ast.Assign) and len(node.targets) == 1:
            if isinstance(node.targets[0], ast.Name) and node.targets[0].id == "Action":
                return {
                    child.id
                    for child in ast.walk(node.value)
                    if isinstance(child, ast.Name) and child.id.endswith("Action")
                }
    fail("could not find the rule Action union")


def keyword_value(call: ast.Call, name: str) -> ast.expr | None:
    for keyword in call.keywords:
        if keyword.arg == name:
            return keyword.value
    return None


def require_keyword(call: ast.Call, name: str, expected: object, context: str) -> None:
    value = keyword_value(call, name)
    if not isinstance(value, ast.Constant) or value.value != expected:
        fail(f"{context} must call model_dump with {name}={expected!r}")


def model_dump_calls(function: ast.AST, receiver_name: str | None = None) -> list[ast.Call]:
    calls: list[ast.Call] = []
    for node in ast.walk(function):
        if not isinstance(node, ast.Call) or not isinstance(node.func, ast.Attribute):
            continue
        if node.func.attr != "model_dump":
            continue
        if receiver_name is not None:
            if not isinstance(node.func.value, ast.Name) or node.func.value.id != receiver_name:
                continue
        calls.append(node)
    return calls


def assigned_model_dump_fields(function: ast.AST) -> dict[str, ast.Call]:
    assignments: dict[str, ast.Call] = {}
    for node in ast.walk(function):
        if not isinstance(node, ast.Assign):
            continue
        calls = [
            candidate
            for candidate in ast.walk(node.value)
            if isinstance(candidate, ast.Call)
            and isinstance(candidate.func, ast.Attribute)
            and candidate.func.attr == "model_dump"
        ]
        if len(calls) != 1:
            continue
        for target in node.targets:
            if isinstance(target, ast.Attribute):
                assignments[target.attr] = calls[0]
    return assignments


def constructor_model_dump_fields(
    function: ast.AST, constructor_name: str
) -> dict[str, ast.Call]:
    for node in ast.walk(function):
        if not isinstance(node, ast.Call) or not isinstance(node.func, ast.Name):
            continue
        if node.func.id != constructor_name:
            continue
        fields: dict[str, ast.Call] = {}
        for keyword in node.keywords:
            calls = [
                candidate
                for candidate in ast.walk(keyword.value)
                if isinstance(candidate, ast.Call)
                and isinstance(candidate.func, ast.Attribute)
                and candidate.func.attr == "model_dump"
            ]
            if keyword.arg is not None and len(calls) == 1:
                fields[keyword.arg] = calls[0]
        return fields
    fail(f"could not find {constructor_name} constructor call")


def require_function_call(function: ast.AST, function_name: str) -> None:
    for node in ast.walk(function):
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Name):
            if node.func.id == function_name:
                return
    fail(f"{getattr(function, 'name', '<function>')} must call {function_name}")


def verify_rule_actions(stable_root: Path, revision: str) -> None:
    schemas = parse_source(
        stable_root, revision, "peakhour/reverseproxy/rules/api/schemas.py"
    )
    require_constant(schemas, "FirewallAction", "reason", None)

    expected_action_classes = {
        "RequestHeadersAction",
        "VConfAction",
        "FirewallAction",
        "RedirectAction",
        "CacheAction",
        "RateLimitRequestAction",
        "RateLimitRequestLateAction",
        "RateLimitResponseAction",
        "RequestRewriteAction",
        "OriginSelectionAction",
        "EarlyHintsSendAction",
    }
    actual_action_classes = action_union_classes(schemas)
    if actual_action_classes != expected_action_classes:
        fail(
            "rule Action union changed: "
            f"got {sorted(actual_action_classes)}, want {sorted(expected_action_classes)}"
        )
    require_optional_fields_default_none(schemas, actual_action_classes)

    models = parse_source(
        stable_root, revision, "peakhour/reverseproxy/rules/models.py"
    )
    add_actions = find_definition(models, ast.FunctionDef, "add_actions_to_rule")
    calls = model_dump_calls(add_actions, "new_action")
    if not calls:
        fail("add_actions_to_rule has no new_action.model_dump calls")
    for call in calls:
        require_keyword(call, "exclude_unset", True, "add_actions_to_rule")
        require_keyword(call, "exclude_defaults", True, "add_actions_to_rule")


def verify_waf_custom_rules(stable_root: Path, revision: str) -> None:
    schemas = parse_source(
        stable_root, revision, "peakhour/reverseproxy/waf/api/schemas.py"
    )
    require_constant(schemas, "WafCustomRuleExpression", "variable_quote_type", None)
    for field_name in ("message", "severity", "tags"):
        require_constant(schemas, "WafLogging", field_name, None)

    # Negative control: blanket null equivalence remains unsafe for action_json.
    require_constant(schemas, "WafAction", "action_arg_val_param_val", "")

    helpers = parse_source(
        stable_root, revision, "peakhour/reverseproxy/helpers.py"
    )

    add_rule = find_definition(helpers, ast.FunctionDef, "add_custom_waf_rule")
    create_fields = constructor_model_dump_fields(add_rule, "WafCustomRule")
    for field_name in ("rules", "action", "logging"):
        call = create_fields.get(field_name)
        if call is None:
            fail(f"add_custom_waf_rule no longer stores {field_name} with model_dump")
        require_keyword(call, "mode", "json", f"add_custom_waf_rule.{field_name}")

    update_rule = find_definition(helpers, ast.FunctionDef, "update_custom_waf_rule")
    assignments = assigned_model_dump_fields(update_rule)
    for field_name in ("rules", "action", "logging"):
        call = assignments.get(field_name)
        if call is None:
            fail(f"update_custom_waf_rule no longer stores {field_name} with model_dump")
        require_keyword(call, "mode", "json", f"update_custom_waf_rule.{field_name}")

    # Negative control: OWASP null currently conflicts with the provider's clear contract.
    set_owasp = find_definition(helpers, ast.FunctionDef, "set_owasp_settings")
    calls = model_dump_calls(set_owasp, "owasp")
    if len(calls) != 1:
        fail("set_owasp_settings must have one owasp.model_dump call")
    require_keyword(calls[0], "exclude_none", True, "set_owasp_settings")


def verify_image_transform_exclusion(stable_root: Path, revision: str) -> None:
    schemas = parse_source(
        stable_root,
        revision,
        "peakhour/domain/image_transforms/api/schemas.py",
    )
    canonicalize = find_definition(schemas, ast.FunctionDef, "canonicalize_preset_config")
    calls = model_dump_calls(canonicalize, "validated")
    if not calls:
        fail("canonicalize_preset_config has no validated.model_dump calls")
    if not any(
        isinstance(keyword_value(call, "exclude_none"), ast.Constant)
        and keyword_value(call, "exclude_none").value is True  # type: ignore[union-attr]
        for call in calls
    ):
        fail("canonicalize_preset_config must exclude nulls during canonical output")
    require_function_call(canonicalize, "_merge_unknown_fields")


def git_head(repository: Path) -> str:
    result = subprocess.run(
        ["git", "-C", str(repository), "rev-parse", "HEAD"],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def expect_contract_failure(check: object, description: str) -> None:
    try:
        assert callable(check)
        check()
    except AssertionError:
        return
    fail(f"self-test mutation was not detected: {description}")


def self_test() -> None:
    valid_schema = ast.parse(
        """
class FirewallAction:
    reason: Optional[str] = None

class OtherAction:
    optional_value: Optional[str] = Field(None, description="optional")

Action = Annotated[Union[FirewallAction, OtherAction], Field(discriminator="type")]
"""
    )
    require_constant(valid_schema, "FirewallAction", "reason", None)
    require_optional_fields_default_none(
        valid_schema, {"FirewallAction", "OtherAction"}
    )
    if action_union_classes(valid_schema) != {"FirewallAction", "OtherAction"}:
        fail("self-test could not read action union classes")

    expect_contract_failure(
        lambda: require_constant(
            ast.parse("class FirewallAction:\n    reason: Optional[str] = ''\n"),
            "FirewallAction",
            "reason",
            None,
        ),
        "nullable default changed",
    )
    expect_contract_failure(
        lambda: require_optional_fields_default_none(
            ast.parse("class FirewallAction:\n    reason: Optional[str] = ''\n"),
            {"FirewallAction"},
        ),
        "optional action field stopped defaulting to None",
    )

    persistence = ast.parse(
        "def store(action):\n"
        "    return action.model_dump(exclude_unset=True, exclude_defaults=True)\n"
    )
    call = model_dump_calls(persistence, "action")[0]
    require_keyword(call, "exclude_unset", True, "self-test persistence")
    require_keyword(call, "exclude_defaults", True, "self-test persistence")
    mutated_persistence = ast.parse(
        "def store(action):\n    return action.model_dump(exclude_unset=True)\n"
    )
    mutated_call = model_dump_calls(mutated_persistence, "action")[0]
    expect_contract_failure(
        lambda: require_keyword(
            mutated_call, "exclude_defaults", True, "self-test persistence"
        ),
        "persistence stopped excluding defaults",
    )


def main(arguments: Iterable[str]) -> int:
    args = list(arguments)
    if args == ["--self-test"]:
        self_test()
        print("stable JSON contract verifier self-test passed")
        return 0
    if len(args) > 1:
        print(
            f"usage: {Path(sys.argv[0]).name} [--self-test|PEAKHOUR_WEBSITE_STABLE_PATH]",
            file=sys.stderr,
        )
        return 2

    default_root = PROVIDER_ROOT.parent / "peakhour-website-stable"
    stable_root = Path(
        args[0]
        if args
        else os.environ.get("PEAKHOUR_WEBSITE_STABLE_PATH", str(default_root))
    ).resolve()
    if not stable_root.is_dir():
        fail(f"stable API repository is unavailable: {stable_root}")

    contract = json.loads(CONTRACT_PATH.read_text(encoding="utf-8"))
    expected_commit = contract["verified_stable_commit"]
    actual_commit = git_head(stable_root)
    if actual_commit != expected_commit:
        fail(
            "stable API checkout does not match the reviewed contract commit: "
            f"got {actual_commit}, want {expected_commit}"
        )

    verify_rule_actions(stable_root, expected_commit)
    verify_waf_custom_rules(stable_root, expected_commit)
    verify_image_transform_exclusion(stable_root, expected_commit)
    print(f"stable JSON contract verified at {actual_commit}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main(sys.argv[1:]))
    except (AssertionError, OSError, subprocess.CalledProcessError) as error:
        print(f"stable JSON contract verification failed: {error}", file=sys.stderr)
        raise SystemExit(1)
