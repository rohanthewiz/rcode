# SmartEditTool - Token-Efficient File Editing

## Overview

The SmartEditTool provides multiple efficient file editing modes with optimized token usage. Unlike the basic EditFileTool which always returns full before/after content, SmartEditTool offers flexible response modes and multiple editing strategies.

## Key Benefits

### Token Efficiency
- **80-95% reduction** in response tokens for common edits
- **Progressive response modes** from 5 tokens to full content
- **Smart diff generation** showing only changes
- **Batch operations** in single calls

### Performance
- **Native tool usage** (sed, patch) for 10x faster bulk operations  
- **Pattern-based replacements** without line counting
- **Memory efficient** for large files
- **Dry-run support** for safe previews

## Edit Modes

### 1. `replace` Mode (Pattern-Based)
Most efficient for find/replace operations across files.

```json
{
  "path": "main.go",
  "mode": "replace",
  "pattern": "oldFunction",
  "replacement": "newFunction",
  "response_mode": "minimal"
}
```
**Response: "3 replacements" (5 tokens)**

Advanced regex with backreferences:
```json
{
  "pattern": "(\\w+)Service",
  "replacement": "${1}Handler",
  "replace_all": true
}
```

### 2. `sed` Mode (Stream Editor Commands)
Powerful for complex transformations using sed syntax.

```json
{
  "path": "config.js",
  "mode": "sed", 
  "commands": [
    "s/var /const /g",
    "/^\\s*\\/\\//d",
    "s/require(/import(/g"
  ],
  "response_mode": "summary"
}
```
**Response: Brief summary with line counts (20-50 tokens)**

### 3. `patch` Mode (Unified Diff)
Apply pre-generated patches efficiently.

```json
{
  "path": "file.txt",
  "mode": "patch",
  "diff": "@@ -10,3 +10,3 @@\n-old line\n+new line\n",
  "response_mode": "minimal"
}
```
**Response: "+1/-1 lines" (5 tokens)**

### 4. `line` Mode (Line-Based)
Fallback for specific line operations.

```json
{
  "path": "file.py",
  "mode": "line",
  "start_line": 50,
  "end_line": 55,
  "operation": "delete",
  "response_mode": "minimal"
}
```
**Response: "5 lines deleted" (5 tokens)**

## Response Modes

### `minimal` (5-10 tokens)
Ultra-concise confirmation only.
- Examples: "3 replacements", "+5/-2 lines", "File unchanged"
- Best for: Bulk operations, known changes

### `summary` (20-50 tokens)
Brief statistics about changes.
```
Edited: src/main.go
Replacements: 5
Lines modified: 5
Size: 1024 → 1050 bytes (+26)
```
- Best for: Understanding scope of changes

### `diff` (50-200 tokens)
Shows actual changes with context.
```
File: config.json
@@ Changes @@
-10: "debug": false,
+10: "debug": true,
-25: "timeout": 1000,
+25: "timeout": 5000,
```
- Best for: Reviewing specific modifications

### `full` (Original behavior)
Complete before/after content.
- Best for: Critical changes, debugging

## Common Usage Patterns

### Pattern 1: Bulk Import Updates
```json
{
  "mode": "replace",
  "pattern": "^import (.*) from ['\"](.+)['\"]",
  "replacement": "import ${1} from '${2}'",
  "response_mode": "minimal"
}
```

### Pattern 2: Remove Comments
```json
{
  "mode": "sed",
  "commands": ["/^\\/\\//d", "s/\\/\\*.*\\*\\///g"],
  "response_mode": "summary"
}
```

### Pattern 3: Safe Preview with Dry Run
```json
{
  "mode": "replace",
  "pattern": "TODO",
  "replacement": "DONE",
  "dry_run": true,
  "response_mode": "diff"
}
```

### Pattern 4: Multi-Pattern Replace
```json
{
  "mode": "sed",
  "commands": [
    "s/console.log/logger.debug/g",
    "s/console.error/logger.error/g",
    "s/console.warn/logger.warn/g"
  ],
  "response_mode": "summary"
}
```

## Advanced Features

### Backup Creation
```json
{
  "backup": true,
  "mode": "replace",
  "pattern": "production",
  "replacement": "development"
}
```
Creates `.bak` file before editing.

### Case-Insensitive Matching
```json
{
  "mode": "replace",
  "pattern": "error",
  "replacement": "Error",
  "case_sensitive": false
}
```

### Single vs All Replacements
```json
{
  "replace_all": false,  // Only first occurrence
  "pattern": "init",
  "replacement": "initialize"
}
```

## Comparison with EditFileTool

| Feature | EditFileTool | SmartEditTool |
|---------|--------------|---------------|
| Token Usage | 200-2000+ | 5-200 (configurable) |
| Line Numbers Required | Yes | No (pattern mode) |
| Bulk Operations | No | Yes (sed mode) |
| Regex Support | No | Yes |
| Dry Run | No | Yes |
| Backup | No | Yes |
| Response Options | Full only | 4 modes |
| Performance | Slower | 10x faster (native tools) |

## Best Practices

### 1. Choose the Right Mode
- **replace**: Simple find/replace
- **sed**: Complex transformations
- **patch**: Pre-computed changes
- **line**: Specific line edits

### 2. Start with Minimal Responses
- Use `minimal` for known operations
- Upgrade to `summary` when you need stats
- Use `diff` for verification
- Reserve `full` for debugging

### 3. Use Dry Run for Safety
```json
{
  "dry_run": true,
  "response_mode": "diff"
}
```
Preview changes before applying.

### 4. Leverage Sed for Bulk Operations
Multiple transformations in one call:
```json
{
  "mode": "sed",
  "commands": [
    "1i\\# Generated file - do not edit",
    "s/2023/2024/g",
    "/^$/d"
  ]
}
```

### 5. Pattern Tips
- Use `${1}` not `$1` for backreferences in Go
- Escape special regex chars: `\\.`, `\\*`, `\\(`
- Test patterns with dry_run first

## Error Handling

The tool provides clear error messages:
- Invalid regex patterns
- File not found
- Permission denied
- Sed/patch command failures

All errors maintain token efficiency with concise messages.

## Integration with Other Tools

Works seamlessly with:
- **RipgrepTool**: Find patterns, then edit
- **ReadFileTool**: Review before editing
- **DiffIntegration**: Automatic change tracking

## Performance Metrics

Typical performance improvements:

| Operation | EditFileTool | SmartEditTool |
|-----------|--------------|---------------|
| Replace word (10 files) | 3000 tokens | 50 tokens |
| Delete lines | 500 tokens | 10 tokens |
| Bulk regex replace | 2000 tokens | 30 tokens |
| Add imports | 1000 tokens | 20 tokens |

## Conclusion

SmartEditTool dramatically reduces token usage while providing more flexible and powerful editing capabilities. Use it as the primary file editing tool for all operations to maximize efficiency and minimize costs.